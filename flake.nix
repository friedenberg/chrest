{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/fea3b367d61c1a6592bc47c72f40a9f3e6a53e96";
    nixpkgs-master.url = "github:NixOS/nixpkgs/b28c4999ed71543e71552ccfd0d7e68c581ba7e9";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };

    bob.url = "github:amarbel-llc/bob";
    tommy.url = "github:amarbel-llc/tommy";
  };

  outputs =
    {
      self,
      nixpkgs,
      nixpkgs-master,
      utils,
      gomod2nix,
      bob,
      tommy,
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
            (final: prev: {
              web-ext = prev.buildNpmPackage rec {
                pname = "web-ext";
                version = "10.1.0";
                src = prev.fetchFromGitHub {
                  owner = "mozilla";
                  repo = "web-ext";
                  rev = version;
                  hash = "sha256-iyhiMX8Qey2VdjIxQnU/YVN3XGwK3uE0JXOV//6dbAc=";
                };
                npmDepsHash = "sha256-z6bE1j8EuEIYKi6bRkAX6KULVShUoXMOQStBX+1QNqk=";
                npmBuildFlags = [ "--production" ];
                passthru.tests.help = prev.runCommand "${pname}-tests" { } ''
                  ${final.web-ext}/bin/web-ext --help
                  touch $out
                '';
                meta = {
                  description = "Command line tool to help build, run, and test web extensions";
                  homepage = "https://github.com/mozilla/web-ext";
                  license = prev.lib.licenses.mpl20;
                  mainProgram = "web-ext";
                };
              };
            })
          ];
        };
        chrest = pkgs-master.buildGoApplication {
          pname = "chrest";
          version = "0.0.1";
          src = ./go;
          subPackages = [
            "cmd/chrest"
            "cmd/chrest-server"
          ];
          modules = ./go/gomod2nix.toml;
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
          packages = [
            bob.packages.${system}.batman
            tommy.packages.${system}.default
          ] ++ (
            with pkgs;
            [
              fish
              gnumake
              jq
              just
              nodejs_latest
              vhs
              zip
            ]
          ) ++ (
            pkgs.lib.optionals pkgs.stdenv.isLinux [
              pkgs.firefox
            ]
          ) ++ (
            with pkgs-master;
            [
              bats
              delve
              go_1_26
              gofumpt
              golangci-lint
              golines
              gopls
              gotools
              govulncheck
              httpie
              nodePackages.bash-language-server
              parallel
              shellcheck
              shfmt
              web-ext
            ]
          ) ++ [
            gomod2nix.packages.${system}.default
          ];
        };
      }
    ));
}
