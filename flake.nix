{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix.url = "github:nix-community/gomod2nix";
  };
  outputs = { self, nixpkgs, flake-utils, gomod2nix }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = import nixpkgs {
            inherit system;
            overlays = [ gomod2nix.overlays.default ];
          };
          packageName = "qperf-go";
        in rec
        {
          packages.default = pkgs.buildGoApplication {
            pname = "${packageName}";
            version = "0.1";
            pwd = ./.;
            src = ./.;
            modules = ./gomod2nix.toml;
          };
          devShells.default = pkgs.mkShell rec {
            buildInputs = with pkgs; [
              pkgs.gomod2nix
            ];
            shellHook = ''
                export LD_LIBRARY_PATH="${pkgs.lib.makeLibraryPath buildInputs}:$LD_LIBRARY_PATH"
                export LD_LIBRARY_PATH="${pkgs.stdenv.cc.cc.lib.outPath}/lib:$LD_LIBRARY_PATH"
                alias sudo='sudo env PATH=$PATH LD_LIBRARY_PATH=$LD_LIBRARY_PATH'
            '';
          };
        }
      );
}
