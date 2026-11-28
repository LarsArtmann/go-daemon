{
  description = "go-daemon — production-proven building blocks for superb unix-socket daemons";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };

  };

  outputs =
    inputs@{
      self,
      flake-parts,
      ...
    }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      # Inline systems: nixpkgs 26.11 dropped x86_64-darwin (which
      # github:nix-systems/default still lists), so evaluating that system
      # fails `nix flake check --all-systems`. aarch64-darwin stays — the
      # library cross-compiles and evaluates there.
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "aarch64-darwin"
      ];

      perSystem =
        {
          pkgs,
          lib,
          ...
        }:
        let
          # Go 1.27.4, not nixpkgs' go_1_27 (1.27.3 at the time of writing):
          # 1.27.3 and earlier carry CVE-2026-97054 / GO-2026-6859 in
          # net/http, which reaches nearly all of socket.go, so the devShell
          # and the release gate must not build with it. The override swaps
          # the source tarball; drop it once nixpkgs ships go_1_27 at 1.27.4
          # or newer (compare `nix eval nixpkgs#go_1_27.version` with
          # `go version` in the devShell).
          goPkg = pkgs.go_1_27.overrideAttrs (_: {
            version = "1.27.4";
            src = pkgs.fetchurl {
              url = "https://go.dev/dl/go1.27.4.src.tar.gz";
              hash = "sha256-c8+4VxrsDAEXdofbhxY9Rqog6XeWcncokaoXZU2aYJk=";
            };
          });

          # All dependencies are public and resolve via the Go proxy, so no
          # mkPreparedSource is needed. GOTOOLCHAIN=local keeps go on the
          # nixpkgs toolchain, GOWORK=off ignores any workspace file up the
          # tree.
          goEnv = {
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
              vendorHash = "sha256-LHju8yvGTXm8lgFJKotvPqWZJUpSj/tiK8EUisuHJws=";
              proxyVendor = false;
              env = {
                inherit GOWORK GOTOOLCHAIN;
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
                  export GOWORK="${GOWORK}"
                  export GOTOOLCHAIN="${GOTOOLCHAIN}"
                  ${text}
                '';
              }
            );
            meta.description = description;
          };
          # The release gate: every check CI runs on a push, in one command
          # with one verdict. It ships in the devShell and NOT as a flake
          # app on purpose: `nix run` chdirs into the store copy of the
          # flake, which has no git history for the docs battery and cannot
          # report the checkout it judged. Logic lives in
          # scripts/release-check.sh; run it with `nix develop -c release-check`.
          gate = pkgs.writeShellApplication {
            name = "release-check";
            runtimeInputs = [
              goPkg
              pkgs.golangci-lint
              pkgs.govulncheck
              pkgs.actionlint
              pkgs.dprint
              pkgs.python3
            ];
            text = with goEnv; ''
              export GOWORK="${GOWORK}"
              export GOTOOLCHAIN="${GOTOOLCHAIN}"
              ${lib.getExe pkgs.bash} ${./scripts/release-check.sh}
            '';
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
                gate
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
