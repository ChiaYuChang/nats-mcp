{ lib, stdenvNoCC, fetchurl }:

let
  version = "0.1.0";
  releases = {
    x86_64-linux = {
      platform = "linux-amd64";
      hash = "sha256-YQ+AfxFBwh0+Z5oWJroAFjJAmMWboZClXxj6q+cBUSk=";
    };
    aarch64-linux = {
      platform = "linux-arm64";
      hash = "sha256-bR1/NxmTirlWTs4EPizhb/faRlHIezjt4S2q+HDahRQ=";
    };
    x86_64-darwin = {
      platform = "darwin-amd64";
      hash = "sha256-4AOUsWaAeboVn+0L/nKplq/jtQ4DioWABYiMlY5uXeA=";
    };
    aarch64-darwin = {
      platform = "darwin-arm64";
      hash = "sha256-qxAR5vyliRP/8QrshumMNJKPtRNgzdUYzsJU7cSgx1c=";
    };
  };
  release = releases.${stdenvNoCC.hostPlatform.system};
in
stdenvNoCC.mkDerivation {
  pname = "nats-mcp";
  inherit version;

  src = fetchurl {
    url = "https://github.com/ChiaYuChang/nats-mcp/releases/download/v${version}/nats-mcp-${release.platform}.tar.gz";
    inherit (release) hash;
  };

  sourceRoot = ".";
  dontStrip = true;

  installPhase = ''
    runHook preInstall
    install -Dm755 nats-mcp $out/bin/nats-mcp
    runHook postInstall
  '';

  doInstallCheck = true;
  installCheckPhase = ''
    runHook preInstallCheck
    test "$(NATS_URL=invalid $out/bin/nats-mcp --version)" = "nats-mcp v${version}"
    runHook postInstallCheck
  '';

  meta = {
    description = "MCP server for NATS messaging, JetStream, and Key-Value operations";
    homepage = "https://github.com/ChiaYuChang/nats-mcp";
    mainProgram = "nats-mcp";
    sourceProvenance = [ lib.sourceTypes.binaryNativeCode ];
    platforms = builtins.attrNames releases;
  };
}
