{
  description = "Hackclub's terminal wakatime program to track terminal time in wakatime";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      allSystems = [
        "x86_64-linux" # 64-bit Intel/AMD Linux
        "aarch64-linux" # 64-bit ARM Linux
        "x86_64-darwin" # 64-bit Intel macOS
        "aarch64-darwin" # 64-bit ARM macOS
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs allSystems (system: f {
        pkgs = import nixpkgs { inherit system; };
      });
    in
    {
      packages = forAllSystems ({ pkgs }: {
        default = pkgs.buildGoModule {
          pname = "terminal-wakatime";
          version = "1.1.5";
          subPackages = [ "cmd/terminal-wakatime" ];
          src = self;
          vendorHash = "sha256-fchZVBY43ccu6nbWn572Qzfgeq4uIwpLf99lOuJCO44=";
        };
      });
    };
}
