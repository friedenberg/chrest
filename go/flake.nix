{
  inputs = {
    nixpkgs-master.url = "github:NixOS/nixpkgs/b28c4999ed71543e71552ccfd0d7e68c581ba7e9";
    nixpkgs.url = "github:NixOS/nixpkgs/23d72dabcb3b12469f57b37170fcbc1789bd7457";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    devenv-go.url = "github:friedenberg/eng?dir=pkgs/alfa/devenv-go";
    devenv-shell.url = "github:friedenberg/eng?dir=pkgs/alfa/devenv-shell";
  };

  outputs =
    {
      self,
      nixpkgs,
      nixpkgs-master,
      utils,
      devenv-go,
      devenv-shell,
    }:
    (utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;

          overlays = [
            devenv-go.overlays.default
          ];
        };

        pkgs-master = import nixpkgs-master {
          inherit system;
          overlays = [
            devenv-go.overlays.default
          ];
        };

        chrest = pkgs-master.buildGoApplication {
          pname = "chrest";
          version = "0.0.1";
          src = ./.;
          subPackages = [
            "cmd/chrest"
            "cmd/chrest-server"
          ];
          modules = ./gomod2nix.toml;
          go = pkgs.go_1_25;
          GOTOOLCHAIN = "local";
        };
      in
      {

        packages.chrest = chrest;
        packages.default = chrest;

        apps.install-mcp = {
          type = "app";
          program = toString (pkgs.writeShellScript "install-chrest-mcp" ''
            set -euo pipefail

            CLAUDE_CONFIG_DIR="''${HOME}/.claude"
            MCP_CONFIG_FILE="''${CLAUDE_CONFIG_DIR}/mcp.json"

            log() {
              ${pkgs.gum}/bin/gum style --foreground 212 "$1"
            }

            log_success() {
              ${pkgs.gum}/bin/gum style --foreground 82 "✓ $1"
            }

            log_error() {
              ${pkgs.gum}/bin/gum style --foreground 196 "✗ $1"
            }

            # Create config directory if needed
            if [[ ! -d "$CLAUDE_CONFIG_DIR" ]]; then
              log "Creating $CLAUDE_CONFIG_DIR..."
              mkdir -p "$CLAUDE_CONFIG_DIR"
            fi

            # Build the flake reference
            FLAKE_REF="${self}"

            # New MCP server entry - runs chrest mcp via nix run
            NEW_SERVER=$(${pkgs.jq}/bin/jq -n \
              --arg cmd "nix" \
              --arg flake "$FLAKE_REF" \
              '{command: $cmd, args: ["run", $flake, "--", "mcp"]}')

            if [[ -f "$MCP_CONFIG_FILE" ]]; then
              log "Found existing MCP config at $MCP_CONFIG_FILE"

              # Check if chrest server already exists
              if ${pkgs.jq}/bin/jq -e '.mcpServers.chrest' "$MCP_CONFIG_FILE" > /dev/null 2>&1; then
                if ${pkgs.gum}/bin/gum confirm "chrest MCP server already configured. Overwrite?"; then
                  UPDATED=$(${pkgs.jq}/bin/jq --argjson server "$NEW_SERVER" '.mcpServers.chrest = $server' "$MCP_CONFIG_FILE")
                  echo "$UPDATED" > "$MCP_CONFIG_FILE"
                  log_success "Updated chrest MCP server configuration"
                else
                  log "Skipping installation"
                  exit 0
                fi
              else
                UPDATED=$(${pkgs.jq}/bin/jq --argjson server "$NEW_SERVER" '.mcpServers.chrest = $server' "$MCP_CONFIG_FILE")
                echo "$UPDATED" > "$MCP_CONFIG_FILE"
                log_success "Added chrest MCP server to existing configuration"
              fi
            else
              log "Creating new MCP config at $MCP_CONFIG_FILE"
              ${pkgs.jq}/bin/jq -n --argjson server "$NEW_SERVER" '{mcpServers: {chrest: $server}}' > "$MCP_CONFIG_FILE"
              log_success "Created MCP configuration"
            fi

            log ""
            log "Installation complete! The chrest MCP server will be available in Claude Code."
            log "Configuration written to: $MCP_CONFIG_FILE"
            log ""
            log "Note: Make sure the chrest browser extension is installed and running."
            log "To verify, run: cat $MCP_CONFIG_FILE"
          '');
        };

        apps.default = {
          type = "app";
          program = "${chrest}/bin/chrest";
        };

        devShells.default = pkgs-master.mkShell {
          packages = (
            with pkgs-master;
            [
              bats
              fish
              gnumake
              just
            ]
          );

          inputsFrom = [
            devenv-go.devShells.${system}.default
            devenv-shell.devShells.${system}.default
          ];
        };
      }
    ));
}
