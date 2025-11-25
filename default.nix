{ pkgs ? import <nixpkgs> { }, lib ? pkgs.lib }:

pkgs.buildGoModule {
  src = ./.;
  name = "cedar";
  version = "0.1.0";
  vendorHash = "sha256-vItryv4JlZ07SjKjstV/QKT3RGCdmjuiPdrifeLvTSA=";

  meta = with lib; {
    description = "Lightweight static site generator.";
    homepage = "https://github.com/ptdewey/cedar";
    license = licenses.mit;
    maintainers = with maintainers; [ ptdewey ];
  };
}
