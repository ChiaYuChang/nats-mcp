package e2e

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

func checkErrorRecovery(stdin io.Writer, scanner *bufio.Scanner) error {
	cases := []struct {
		name string
		args map[string]any
	}{
		{"nonexistent_e2e_tool", map[string]any{}},
		{"nats_publish", map[string]any{"payload": "missing-subject"}},
	}
	for i, test := range cases {
		id := 20 + i*2
		response, err := callTool(stdin, scanner, id, test.name, test.args)
		if err != nil {
			if _, ok := errors.AsType[*rpcError](err); !ok {
				return fmt.Errorf("%s transport failure: %w", test.name, err)
			}
		} else if !response.Result.IsError {
			return fmt.Errorf("%s unexpectedly accepted invalid input", test.name)
		}
		// A valid call after each failure verifies the same process remains usable.
		ping, err := callTool(stdin, scanner, id+1, "nats_ping", map[string]any{})
		if err != nil {
			return fmt.Errorf("ping after %s: %w", test.name, err)
		}
		if !confirmsPing(ping) {
			return fmt.Errorf("ping after %s did not confirm connectivity", test.name)
		}
	}
	return nil
}
