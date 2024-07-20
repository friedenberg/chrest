{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
    utils.url = "github:numtide/flake-utils";

    go = {
      url = "github:friedenberg/dev-flake-templates?dir=go";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    js = {
      url = "github:friedenberg/dev-flake-templates?dir=javascript";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, utils, go, js }:
    (utils.lib.eachDefaultSystem
      (system:
        let

          pkgs = import nixpkgs {
            inherit system;
            overlays = [
              go.overlays.default
            ];
          };

        in
        {
          devShells.default = pkgs.mkShell {
            packages = (with pkgs; [
              fish
              gnumake
            ]);

            inputsFrom = [
              go.devShells.${system}.default
              js.devShells.${system}.default
            ];
          };
        })
    );
}
