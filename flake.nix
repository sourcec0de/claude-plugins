{
  description = "Claude Code plugins that constrain and improve model output quality";

  # nixpkgs is pulled as a channel tarball rather than github:NixOS/nixpkgs so
  # that the flake resolves in sandboxed environments without GitHub API access.
  # The lock file pins a narHash, so this is exactly as reproducible.
  inputs.nixpkgs.url = "https://channels.nixos.org/nixos-unstable/nixexprs.tar.xz";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f:
        nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      commands = [ "astgrep-lint" "bashguard" "autofmt" ];

      # What a contributor needs. Users need none of it: they install the
      # binaries below, which carry their own ast-grep.
      toolchain = pkgs: with pkgs; [ go gopls ast-grep jq nodejs ];

      goEnv = ''
        export HOME="$TMPDIR"
        export GOCACHE="$TMPDIR/go-cache"
        export GOMODCACHE="$TMPDIR/go-mod"
        export GOFLAGS="-mod=vendor"
        export GOPROXY=off
        export CGO_ENABLED=0
      '';

      inTree = ''
        cp -r ${self} tree
        chmod -R u+w tree
        cd tree
      '';
    in
    {
      # The hooks are these binaries. Users install them once —
      #   nix profile install github:sourcec0de/claude-plugins
      # — and the plugin's hook configuration invokes them by name off PATH.
      # Nothing is compiled on a user's machine and nothing needs Go installed.
      packages = forAllSystems (pkgs: rec {
        default = pkgs.buildGoModule {
          pname = "claude-plugins";
          version = "0.1.0";
          src = self;

          # vendor/ is committed, so the build needs no network and no hash to
          # chase when a dependency changes.
          vendorHash = null;

          subPackages = map (c: "cmd/${c}") commands;

          env.CGO_ENABLED = 0;
          ldflags = [ "-s" "-w" ];

          nativeBuildInputs = [ pkgs.makeWrapper ];

          # buildGoModule runs the test suite, and these tests spawn the real
          # ast-grep against the real rule tree rather than a stub.
          nativeCheckInputs = [ pkgs.ast-grep ];

          # The rule trees ship inside the package so the binaries work as
          # standalone commands. When Claude Code runs them as hooks it exports
          # CLAUDE_PLUGIN_ROOT, which takes precedence, so rule changes reach
          # users with a plugin update rather than a reinstall.
          #
          # ast-grep is put on the wrapped PATH rather than left to the user,
          # which is the whole reason these are Nix packages: the runtime
          # dependency is declared, not hoped for.
          postInstall = ''
            mkdir -p $out/share/claude-plugins
            cp -r astgrep bashguard $out/share/claude-plugins/

            for command in ${builtins.concatStringsSep " " commands}; do
              wrapProgram $out/bin/$command \
                --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.ast-grep ]} \
                --set-default CLAUDE_PLUGINS_RULE_ROOT $out/share/claude-plugins
            done
          '';

          meta = {
            description = "Hook binaries for the claude-plugins marketplace";
            mainProgram = "astgrep-lint";
          };
        };
      });

      apps = forAllSystems (pkgs:
        nixpkgs.lib.genAttrs commands (command: {
          type = "app";
          program = "${self.packages.${pkgs.system}.default}/bin/${command}";
        }));

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = toolchain pkgs;
          shellHook = ''
            echo "claude-plugins dev shell: go $(go version | cut -d' ' -f3), ast-grep $(ast-grep --version | cut -d' ' -f2)"
          '';
        };
      });

      checks = forAllSystems (pkgs:
        let
          check = name: script:
            pkgs.runCommand name { nativeBuildInputs = toolchain pkgs; } ''
              ${inTree}
              ${script}
              touch $out
            '';
        in
        {
          # Building the package is itself a check: if the binaries do not
          # build reproducibly there is nothing to install.
          package = self.packages.${pkgs.system}.default;

          # The wrapped binaries must actually run, find ast-grep, and locate
          # their bundled rules with no CLAUDE_PLUGIN_ROOT set. That is exactly
          # the standalone path a user gets after `nix profile install`.
          packaged-binaries =
            pkgs.runCommand "packaged-binaries"
              { nativeBuildInputs = [ self.packages.${pkgs.system}.default ]; } ''
              cat > sample.go <<'GO'
              package sample

              import "fmt"

              func Leak() {
                fmt.Println("debug")
              }
              GO

              astgrep-lint sample.go 2>err.txt && {
                echo "expected a violation to be reported" >&2; exit 1; }
              grep -q no-fmt-println err.txt || {
                echo "expected no-fmt-println; got:" >&2; cat err.txt >&2; exit 1; }

              bashguard 'rm -rf build' 2>bash.txt && {
                echo "expected rm -rf to be flagged" >&2; exit 1; }
              grep -q no-rm-rf bash.txt || {
                echo "expected no-rm-rf; got:" >&2; cat bash.txt >&2; exit 1; }

              touch $out
            '';

          gotest = check "gotest" ''
            ${goEnv}
            export CLAUDE_PLUGIN_ROOT="$PWD"
            go vet ./...
            go test ./...
          '';

          gofmt = check "gofmt" ''
            ${goEnv}
            unformatted="$(gofmt -l cmd hookio)"
            if [ -n "$unformatted" ]; then
              echo "gofmt needed for:" >&2
              echo "$unformatted" >&2
              exit 1
            fi
          '';

          astgrep-rules = check "astgrep-rules" ''
            (cd astgrep   && ast-grep test --config sgconfig.yml)
            (cd bashguard && ast-grep test --config sgconfig.yml)
          '';

          manifests = check "manifests" ''
            for f in .claude-plugin/marketplace.json hooks/*.json; do
              jq -e . "$f" > /dev/null || { echo "invalid JSON: $f" >&2; exit 1; }
            done
          '';
        });
    };
}
