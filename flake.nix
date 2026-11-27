{
  description = "sinkctl: CLI and MCP server for Bugsink";

  # unstable: go.mod needs Go >= 1.26
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      # lastModifiedDate is YYYYMMDDhhmmss; release builds stamp RFC 3339
      d = self.lastModifiedDate or "19700101000000";
      part = start: len: builtins.substring start len d;
      date = "${part 0 4}-${part 4 2}-${part 6 2}T${part 8 2}:${part 10 2}:${part 12 2}Z";

      forAllSystems = nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.buildGoModule {
            pname = "sinkctl";
            # semver comes from release tags; flake builds report dev plus the exact commit
            version = "dev";
            src = self;
            subPackages = [ "cmd/sinkctl" ];
            vendorHash = "sha256-mjUEBtyhrCvoTMyW43aBpY35aYFxRcIFxVZYU1auCK4=";
            env.CGO_ENABLED = 0;
            ldflags = [
              "-s"
              "-w"
              "-X github.com/kidk/sinkctl/internal/version.Commit=${self.rev or self.dirtyRev or "unknown"}"
              "-X github.com/kidk/sinkctl/internal/version.Date=${date}"
            ];
            meta = {
              description = "CLI and MCP server for Bugsink";
              homepage = "https://github.com/kidk/sinkctl";
              license = nixpkgs.lib.licenses.asl20;
              mainProgram = "sinkctl";
            };
          };
        }
      );

      overlays.default = final: _prev: {
        sinkctl = self.packages.${final.stdenv.hostPlatform.system}.default;
      };
    };
}
