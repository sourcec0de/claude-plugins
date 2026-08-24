{
  description = "Personal Claude Code plugin marketplace: ast-grep enforcement, shell guardrails, and workflow commands";

  # nixpkgs is pulled as a channel tarball rather than github:NixOS/nixpkgs so
  # that the flake resolves in sandboxed environments without GitHub API access.
  # The lock file pins a narHash, so this is exactly as reproducible.
  inputs.nixpkgs.url = "https://channels.nixos.org/nixos-unstable/nixexprs.tar.xz";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f:
        nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      # Everything a check or a contributor needs. The devShell and the checks
      # deliberately share one list so that `nix develop` reproduces CI.
      toolchain = pkgs: with pkgs; [ go gopls ast-grep jq shellcheck nodejs ];

      # Go inside the build sandbox has no network, no writable home and no C
      # compiler, and the hook tests shell out to ast-grep, so every check that
      # touches Go sets these up the same way. Nothing here needs cgo, so it is
      # disabled rather than dragging gcc into the closure.
      goEnv = ''
        export HOME="$TMPDIR"
        export GOCACHE="$TMPDIR/go-cache"
        export GOMODCACHE="$TMPDIR/go-mod"
        export GOFLAGS="-mod=vendor"
        export GOPROXY=off
        export CGO_ENABLED=0
      '';

      # Checks run against a writable copy of the tree: ast-grep writes nothing,
      # but Go wants a mutable working directory.
      inTree = ''
        cp -r ${self} tree
        chmod -R u+w tree
        cd tree
      '';
    in
    {
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
          # The hook tests spawn the real ast-grep against the real rule tree,
          # so CLAUDE_PLUGIN_ROOT must point at the copied source.
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

          shellcheck = check "shellcheck" ''
            shellcheck bin/claude-hooks
          '';

          manifests = check "manifests" ''
            for f in .claude-plugin/marketplace.json hooks/*.json; do
              jq -e . "$f" > /dev/null || { echo "invalid JSON: $f" >&2; exit 1; }
            done
          '';
        });
    };
}
