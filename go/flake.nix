{
  inputs = {
    nixpkgs-master.url = "github:NixOS/nixpkgs/fa83fd837f3098e3e678e6cf017b2b36102c7211";
    nixpkgs.url = "github:NixOS/nixpkgs/54b154f971b71d260378b284789df6b272b49634";
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
        };

        chrest = pkgs-master.buildGoApplication {
          pname = "chrest";
          version = "0.0.1";
          src = ./.;
          subPackages = [
            "cmd/chrest"
          ];
          modules = ./gomod2nix.toml;
          go = pkgs.go_1_25;
          GOTOOLCHAIN = "local";
        };
      in
      {

        packages.chrest = chrest;
        packages.default = chrest;

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
