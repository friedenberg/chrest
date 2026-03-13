{
  inputs = {
    nixpkgs-master.url = "github:NixOS/nixpkgs/b28c4999ed71543e71552ccfd0d7e68c581ba7e9";
    nixpkgs.url = "github:NixOS/nixpkgs/23d72dabcb3b12469f57b37170fcbc1789bd7457";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    devenv-go.url = "github:amarbel-llc/purse-first?dir=devenvs/go";
    devenv-shell.url = "github:amarbel-llc/purse-first?dir=devenvs/shell";
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
