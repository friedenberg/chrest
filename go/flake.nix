{
  inputs = {
    nixpkgs-master.url = "github:NixOS/nixpkgs/b28c4999ed71543e71552ccfd0d7e68c581ba7e9";
    nixpkgs.url = "github:NixOS/nixpkgs/23d72dabcb3b12469f57b37170fcbc1789bd7457";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      nixpkgs-master,
      utils,
      gomod2nix,
    }:
    (utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            gomod2nix.overlays.default
          ];
        };

        pkgs-master = import nixpkgs-master {
          inherit system;
          overlays = [
            gomod2nix.overlays.default
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
          go = pkgs-master.go_1_26;
          GOTOOLCHAIN = "local";
          postInstall = ''
            $out/bin/chrest generate-plugin $out
          '';
        };
      in
      {

        packages.chrest = chrest;
        packages.default = chrest;

        apps.default = {
          type = "app";
          program = "${chrest}/bin/chrest";
        };

        devShells.default = pkgs-master.mkShell {
          packages = (
            with pkgs-master;
            [
              bats
              delve
              fish
              gnumake
              go_1_26
              gofumpt
              golangci-lint
              golines
              gopls
              gotools
              govulncheck
              just
              nodePackages.bash-language-server
              parallel
              shellcheck
              shfmt
            ]
          ) ++ [
            gomod2nix.packages.${system}.default
          ];
        };
      }
    ));
}
