package e2e

import (
	"bufio"
	"context"
	"encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/nats-mcp/nats-mcp/internal/testutil"
)

const (
	protocolVersion = "2024-11-05"
	e2eToken        = "e2e-token"
)

type rpcRequest struct {
	JSONRPC string "json:\"jsonrpc\""
	ID      int    "json:\"id,omitzero\""
	Method  string "json:\"method\""
	Params  any    "json:\"params,omitempty\""
}

type rpcResponse struct {
	JSONRPC string    "json:\"jsonrpc\""
	ID      int       "json:\"id\""
	Result  rpcResult "json:\"result\""
	Error   *rpcError "json:\"error\""
}

type rpcResult struct {
	ProtocolVersion string       "json:\"protocolVersion\""
	Tools           []rpcTool    "json:\"tools\""
	Content         []rpcContent "json:\"content\""
	IsError         bool         "json:\"isError\""
}

type rpcTool struct {
	Name string "json:\"name\""
}

type rpcContent struct {
	Type string "json:\"type\""
	Text string "json:\"text\""
}

type rpcError struct {
	Code    int    "json:\"code\""
	Message string "json:\"message\""
}

var binaryPath = flag.String("binary", "", "path to an existing nats-mcp binary")

func TestMCPStdio(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	if err := run(ctx, *binaryPath, t); err != nil {
		t.Fatal(err)
	}
}

func run(ctx context.Context, binaryPath string, t *testing.T) error {
	binaryPath, cleanupBinary, err := prepareBinary(ctx, binaryPath)
	if err != nil {
		return err
	}
	defer cleanupBinary()

	container, err := testutil.SetupNATS(ctx, e2eToken)
	if err != nil {
		return fmt.Errorf("start NATS test container: %w", err)
	}
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := container.Terminate(cleanupCtx); err != nil {
			log.Printf("terminate NATS test container: %v", err)
		}
	}()

	if err := exerciseMCP(ctx, binaryPath, container.URL, e2eToken, t); err != nil {
		return err
	}

	t.Log("E2E passed: initialization, tools, messaging, error recovery, graceful shutdown")
	return nil
}

func prepareBinary(ctx context.Context, binaryPath string) (string, func(), error) {
	if binaryPath != "" {
		absolutePath, err := filepath.Abs(binaryPath)
		if err != nil {
			return "", nil, fmt.Errorf("resolve binary path: %w", err)
		}
		info, err := os.Stat(absolutePath)
		if err != nil {
			return "", nil, fmt.Errorf("stat server binary: %w", err)
		}
		if info.IsDir() {
			return "", nil, fmt.Errorf("server binary path %q is a directory", absolutePath)
		}
		return absolutePath, func() {}, nil
	}

	if _, err := os.Stat("../go.mod"); err != nil {
		return "", nil, errors.New("run go test ./e2e or pass -args -binary")
	}

	tempDir, err := os.MkdirTemp("", "nats-mcp-e2e-")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary build directory: %w", err)
	}

	name := "nats-mcp"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binaryPath = filepath.Join(tempDir, name)

	build := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "./cmd/nats-mcp")
	build.Dir = ".."
	output, err := build.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return "", nil, fmt.Errorf("build server binary: %w: %s", err, strings.TrimSpace(string(output)))
	}

	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			log.Printf("remove temporary build directory: %v", err)
		}
	}
	return binaryPath, cleanup, nil
}

func exerciseMCP(ctx context.Context, binaryPath, natsURL, token string, t *testing.T) (resultErr error) {
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Env = serverEnvironment(natsURL, token)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open MCP stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("open MCP stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return fmt.Errorf("start server binary: %w", err)
	}
	defer func() {
		// EOF must terminate the server successfully; kill only after a bounded wait.
		closeErr := stdin.Close()
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			resultErr = errors.Join(resultErr, closeErr, err)
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			waitErr := <-done
			resultErr = errors.Join(resultErr, closeErr, waitErr, errors.New("MCP server did not exit within 5 seconds after stdin EOF"))
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	initialize, err := exchange(stdin, scanner, rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "nats-mcp-e2e",
				"version": "1.0.0",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("initialize MCP server: %w", err)
	}
	if initialize.Result.ProtocolVersion == "" {
		return errors.New("initialize response did not include a protocol version")
	}

	if err := writeJSONLine(stdin, rpcRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}); err != nil {
		return fmt.Errorf("send initialized notification: %w", err)
	}

	tools, err := exchange(stdin, scanner, rpcRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
		Params:  map[string]any{},
	})
	if err != nil {
		return fmt.Errorf("list MCP tools: %w", err)
	}

	hasPing := false
	for _, tool := range tools.Result.Tools {
		if tool.Name == "nats_ping" {
			hasPing = true
			break
		}
	}
	if !hasPing {
		return errors.New("tools/list response did not contain nats_ping")
	}

	ping, err := exchange(stdin, scanner, rpcRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "nats_ping",
			"arguments": map[string]any{},
		},
	})
	if err != nil {
		return fmt.Errorf("call nats_ping: %w", err)
	}
	if ping.Result.IsError {
		return errors.New("nats_ping returned an MCP tool error")
	}
	for _, content := range ping.Result.Content {
		if strings.Contains(content.Text, "NATS Server is reachable") {
			return exerciseFeatures(ctx, stdin, scanner, natsURL, token, t)
		}
	}
	return errors.New("nats_ping response did not confirm NATS connectivity")
}

func exchange(stdin io.Writer, scanner *bufio.Scanner, request rpcRequest) (rpcResponse, error) {
	if err := writeJSONLine(stdin, request); err != nil {
		return rpcResponse{}, err
	}
	return readResponse(scanner, request.ID)
}

func writeJSONLine(writer io.Writer, value rpcRequest) error {
	message, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal MCP message: %w", err)
	}
	message = append(message, '\n')
	for len(message) > 0 {
		written, err := writer.Write(message)
		if err != nil {
			return fmt.Errorf("write MCP message: %w", err)
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		message = message[written:]
	}
	return nil
}

func readResponse(scanner *bufio.Scanner, id int) (rpcResponse, error) {
	for scanner.Scan() {
		var response rpcResponse
		if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
			return rpcResponse{}, fmt.Errorf("decode MCP response: %w", err)
		}
		if response.ID != id {
			continue
		}
		if response.JSONRPC != "2.0" {
			return rpcResponse{}, fmt.Errorf("unexpected JSON-RPC version %q", response.JSONRPC)
		}
		if response.Error != nil {
			return rpcResponse{}, response.Error
		}
		return response, nil
	}
	if err := scanner.Err(); err != nil {
		return rpcResponse{}, fmt.Errorf("read MCP response: %w", err)
	}
	return rpcResponse{}, fmt.Errorf("server closed stdout before response %d: %w", id, io.EOF)
}

func serverEnvironment(natsURL, token string) []string {
	env := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		switch name {
		case "NATS_URL", "NATS_CREDS", "NATS_TOKEN", "NATS_USER", "NATS_PASSWORD":
			continue
		}
		env = append(env, entry)
	}
	return append(env, "NATS_URL="+natsURL, "NATS_TOKEN="+token)
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

func callTool(stdin io.Writer, scanner *bufio.Scanner, id int, name string, args map[string]any) (rpcResponse, error) {
	return exchange(stdin, scanner, rpcRequest{
		JSONRPC: "2.0", ID: id, Method: "tools/call",
		Params: map[string]any{"name": name, "arguments": args},
	})
}

func exerciseFeatures(ctx context.Context, stdin io.Writer, scanner *bufio.Scanner, natsURL, token string, t *testing.T) error {
	checks := []struct {
		name string
		run  func() error
	}{
		{"Messaging", func() error { return checkMessaging(ctx, stdin, scanner, natsURL, token) }},
		{"ErrorRecovery", func() error { return checkErrorRecovery(stdin, scanner) }},
	}
	for _, check := range checks {
		if !t.Run(check.name, func(t *testing.T) {
			if err := check.run(); err != nil {
				t.Fatal(err)
			}
		}) {
			return fmt.Errorf("%s failed", check.name)
		}
	}
	return nil
}
