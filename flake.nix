{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/3e20095fe3c6cbb1ddcef89b26969a69a1570776";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    devenv-go.url = "github:amarbel-llc/purse-first?dir=devenvs/go";
    devenv-browser_extension.url = "path:./devenvs/browser_extension";
  };

  outputs =
    {
      self,
      nixpkgs,
      utils,
      devenv-go,
      devenv-browser_extension,
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
              just
            ]
          );

          inputsFrom = [
            devenv-go.devShells.${system}.default
            devenv-browser_extension.devShells.${system}.default
          ];
        };
      }
    ));
}
