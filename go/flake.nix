{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
    utils.url = "github:numtide/flake-utils";

    go = {
      url = "github:friedenberg/dev-flake-templates?dir=go";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, utils, go }:
    (utils.lib.eachDefaultSystem
      (system:
        let

          pkgs = import nixpkgs {
            inherit system;
            overlays = [
              go.overlays.default
            ];
          };

          chrest = pkgs.buildGoApplication {
            name = "chrest";
            pname = "zit";
            version = "1.5";
            src = ./cmd/chrest;
            modules = ./gomod2nix.toml;
            doCheck = false;
            enableParallelBuilding = true;
          };

        in
        {
          pname = "chrest";
          packages.default = chrest;
          devShells.default = pkgs.mkShell {
            packages = (with pkgs; [
              fish
              gnumake
            ]);

            inputsFrom = [
              go.devShells.${system}.default
            ];
          };
        })
    );
}
