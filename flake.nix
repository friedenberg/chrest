{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/54b154f971b71d260378b284789df6b272b49634";
    nixpkgs-master.url = "github:NixOS/nixpkgs/fa83fd837f3098e3e678e6cf017b2b36102c7211";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    devenv-go.url = "github:friedenberg/eng?dir=pkgs/alfa/devenv-go";
    devenv-js.url = "github:friedenberg/eng?dir=pkgs/alfa/devenv-js";
  };

  outputs =
    {
      self,
      nixpkgs,
      nixpkgs-master,
      utils,
      devenv-go,
      devenv-js,
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
      in
      {
        devShells.default = pkgs.mkShell {
          packages = (
            with pkgs;
            [
              fish
              gnumake
              httpie
              jq
              just
              web-ext
            ]
          );

          inputsFrom = [
            devenv-go.devShells.${system}.default
            devenv-js.devShells.${system}.default
          ];
        };
      }
    ));
}
