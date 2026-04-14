{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/fea3b367d61c1a6592bc47c72f40a9f3e6a53e96";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    bob.url = "github:amarbel-llc/bob";
    devenv-browser_extension.url = "path:./devenvs/browser_extension";
    devenv-go.url = "github:amarbel-llc/purse-first?dir=devenvs/go";
    tommy.url = "github:amarbel-llc/tommy";
  };

  outputs =
    {
      self,
      nixpkgs,
      utils,
      bob,
      devenv-go,
      devenv-browser_extension,
      tommy,
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
          packages = [
            bob.packages.${system}.batman
            tommy.packages.${system}.default
          ] ++ (
            with pkgs;
            [
              fish
              gnumake
              just
              vhs
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
