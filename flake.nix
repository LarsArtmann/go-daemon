{
  description = "go-daemon — production-proven building blocks for superb unix-socket daemons";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

    systems.url = "github:nix-systems/default";
  };

  outputs =
    inputs@{
      self,
      flake-parts,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;

      perSystem =
        {
          pkgs,
          lib,
          ...
        }:
        let
          goPkg = pkgs.go_1_27;

          # All dependencies are public and resolve via the Go proxy; no
          # mkPreparedSource needed (contrast: private-repo consumers like
          # project-discovery-daemon pin this repo as a git+ssh flake input).
          # GOTOOLCHAIN=local keeps go on the nixpkgs toolchain, GOWORK=off
          # ignores any workspace file up the tree.
          goEnv = {
            GOPRIVATE = "github.com/larsartmann/*,github.com/LarsArtmann/*";
            GOWORK = "off";
            GOTOOLCHAIN = "local";
          };

          src = lib.fileset.toSource {
            root = ./.;
            fileset = lib.fileset.unions [
              ./go.mod
              ./go.sum
              ./LICENSE
              (lib.fileset.fileFilter (file: file.hasExt "go") ./.)
            ];
          };

          buildGoModule = pkgs.buildGoModule.override { go = goPkg; };

          library =
            with goEnv;
            buildGoModule {
              pname = "go-daemon";
              version = self.shortRev or self.dirtyShortRev or "dev";
              inherit src;
              vendorHash = "sha256-Y2lZeBxujG/tvcT8XntqRr1VhZRevY1bLYtHt4ekenw=";
              proxyVendor = false;
              env = {
                inherit
                  GOPRIVATE
                  GOWORK
                  GOTOOLCHAIN
                  ;
              };
              doCheck = false; # tests run via `nix run .#test` (race + tmp sockets)

              meta = {
                description = "Production-proven building blocks for superb unix-socket daemons";
                license = lib.licenses.mit;
                maintainers = [
                  {
                    name = "Lars Artmann";
                    github = "LarsArtmann";
                  }
                ];
              };
            };

          mkGoApp = name: description: runtimeInputs: text: {
            type = "app";
            program = lib.getExe (
              pkgs.writeShellApplication {
                inherit name;
                inherit runtimeInputs;
                text = with goEnv; ''
                  export GOPRIVATE="${GOPRIVATE}"
                  export GOWORK="${GOWORK}"
                  export GOTOOLCHAIN="${GOTOOLCHAIN}"
                  ${text}
                '';
              }
            );
            meta.description = description;
          };
        in
        {
          packages = {
            default = library;
            go-daemon = library;
          };

          apps = {
            test = mkGoApp "run-test" "Run the go-daemon test suite" [
              goPkg
            ] "${lib.getExe goPkg} test -race -count=1 ./...";

            lint = mkGoApp "run-lint" "Run golangci-lint across the project" [
              goPkg
              pkgs.golangci-lint
            ] "${lib.getExe pkgs.golangci-lint} run ./...";

            build = mkGoApp "run-build" "Build all go-daemon packages" [
              goPkg
            ] "${lib.getExe goPkg} build ./...";

            # Compiles the first go fence of README.md (the quickstart) so a
            # broken example fails the build (the caf1f0c class). The logic
            # lives in scripts/check-readme.sh — the single source of truth
            # that CI runs directly, so the two gates can never drift.
            check-readme =
              mkGoApp "check-readme" "Compile the README quickstart snippet against the library"
                [
                  goPkg
                  pkgs.bash
                  pkgs.gawk
                ]
                ''
                  ${lib.getExe pkgs.bash} ${./scripts/check-readme.sh}
                '';

            # Documentation health as one command: dprint check, table-row
            # completeness, strike presence in historical reports, and the
            # FEATURES counts derivation. Logic lives in
            # scripts/check-docs.sh so CI can run the identical gate.
            check-docs =
              mkGoApp "check-docs" "Run the documentation health gates (format, rows, strikes, counts)"
                [
                  pkgs.bash
                  pkgs.dprint
                  pkgs.python3
                ]
                ''
                  ${lib.getExe pkgs.bash} ${./scripts/check-docs.sh}
                '';
          };

          checks.build = library;

          devShells.default = pkgs.mkShell (
            {
              packages = [
                goPkg
                pkgs.golangci-lint
                pkgs.govulncheck
                pkgs.gopls
                pkgs.actionlint
                pkgs.dprint
              ];
            }
            // goEnv
          );

          # nixfmt-tree wraps nixfmt for directory formatting (plain nixfmt
          # no longer accepts bare directory/no-arg invocations).
          formatter = pkgs.nixfmt-tree;
        };
    };
}
