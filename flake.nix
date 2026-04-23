{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/fea3b367d61c1a6592bc47c72f40a9f3e6a53e96";
    nixpkgs-master.url = "github:NixOS/nixpkgs/e2dde111aea2c0699531dc616112a96cd55ab8b5";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    gomod2nix = {
      url = "github:amarbel-llc/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "utils";
    };

    bob = {
      url = "github:amarbel-llc/bob";
      inputs.gomod2nix.follows = "gomod2nix";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };

    tommy = {
      url = "github:amarbel-llc/tommy";
      inputs.gomod2nix.follows = "gomod2nix";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.bob.follows = "bob";
    };
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
        firefox = pkgs.callPackage ./nix/firefox.nix { };
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
          nativeBuildInputs = [ pkgs.makeWrapper ];
          postInstall = ''
            $out/bin/chrest generate-plugin $out
          '';
          postFixup =
            let
              monolithBinPath = "${pkgs.monolith}/bin";
            in
            ''
              wrapProgram $out/bin/chrest \
                --prefix PATH : ${firefox}/bin:${monolithBinPath}
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
              unixtools.xxd
              vhs
              zip
            ]
          ) ++ [
            firefox
            pkgs.monolith
          ] ++ (
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

          # Passthru: use the outer-shell git (user's nix profile, NixOS
          # system path, or distro). Respects the user's gitconfig,
          # signing keys, and hooks, and keeps `git` behavior identical
          # inside and outside the devshell. Without this,
          # `just go/build-nix-gomod`'s drift guard fails with
          # `git: command not found` under `nix develop --command`.
          #
          # Only prepends the single directory the located git lives in
          # — avoids polluting PATH with /usr/bin wholesale.
          shellHook = ''
            if ! command -v git >/dev/null 2>&1; then
              for candidate in \
                "$HOME/.nix-profile/bin/git" \
                /run/current-system/sw/bin/git \
                /etc/profiles/per-user/"$USER"/bin/git \
                /usr/bin/git \
                /bin/git; do
                if [ -x "$candidate" ]; then
                  export PATH="$(dirname "$candidate"):$PATH"
                  break
                fi
              done
            fi
          '';
        };
      }
    ));
}
