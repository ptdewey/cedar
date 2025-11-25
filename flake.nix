{
  description = "Cedar flake";
  inputs = { nixpkgs.url = "nixpkgs/nixpkgs-unstable"; };
  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
      cedar = pkgs.callPackage ./default.nix { };
    in {
      packages.${system}.default = cedar;

      legacyPackages.${system}.default = cedar;

      apps.${system}.cedar = {
        type = "app";
        program = "${cedar}/bin/cedar";
      };

      defaultPackage.${system} = cedar;
    };
}
