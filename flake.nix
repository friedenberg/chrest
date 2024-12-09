{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    nixpkgs-stable.url = "github:NixOS/nixpkgs/nixos-24.05";
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

  outputs = { self, nixpkgs, nixpkgs-stable, utils, go, js }:
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
            pname = "chrest";
            version = "0.0.1";
            pwd = ./go/cmd;
            src = ./go/cmd;
            modules = ./go/cmd/gomod2nix.toml;
            doCheck = false;
            enableParallelBuilding = true;
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

          pname = "chrest";
          packages.default = chrest;
        })
    );
}
