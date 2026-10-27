{
  description = "Flows dev env";

  inputs = {
    # Published stable Go compiler and matching module builder only.
    nixpkgs-go-stable = {
      url = "https://api.flakehub.com/f/pinned/NixOS/nixpkgs/0.2605.1016949%2Brev-1f607aa79b6356ed45d3cd3c8079c196e79efa2c/01a17424-ae2d-7b20-83c4-cf14527294c3/source.tar.gz";
      flake = false;
    };

    # Retain golangci-lint 2.15.0; stable currently provides 2.12.2.
    go-tooling = {
      url = "github:NixOS/nixpkgs/8cec5ef1131078ea76689e84c362b0ed6a1fb6a1";
      flake = false;
    };
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.1.*.tar.gz";

    goreleaser = {
      url = "github:goreleaser/nur/52282e9a2d7c6882e9e0e8675d593b928c89e323";
      flake = false;
    };
  };

  outputs = { go-tooling, self, nixpkgs, nixpkgs-go-stable, goreleaser }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forEachSupportedSystem = f:
        nixpkgs.lib.genAttrs supportedSystems (system:
          let
            pkgs = import nixpkgs {
              inherit system;
              overlays = [ self.overlays.default  ];
              config.allowUnfree = true;
            };
          in
          f { pkgs = pkgs; system = system; }
        );

      speakeasyVersion = "1.351.0";
      speakeasyPlatforms = {
        "x86_64-linux"   = "linux_amd64";
        "aarch64-linux"  = "linux_arm64";
        "x86_64-darwin"  = "darwin_amd64";
        "aarch64-darwin" = "darwin_arm64";
      };
      speakeasyHashes = {
        "x86_64-linux"   = "sha256-eeGzoIlsgsW/wnEET+UFgNN1qWgndxRHzulhHDTyHRY=";
        "aarch64-linux"  = "sha256-zOj2QUwLwRz0MyTLdVxLaGU7XEqhgKLCyhsO9S8VCNk=";
        "x86_64-darwin"  = "sha256-vBgEv6WWwJhBW6nMLy4Nj7qjWdGqk/4al5dIUCqrm1I=";
        "aarch64-darwin" = "sha256-N129T0BDRVUXxH6Dl58/hUEApiq1q2B6qTwTuEjLDi4=";
      };

    in
    {
      overlays.default = final: prev:
        let
          goPkgs = import nixpkgs-go-stable { system = prev.stdenv.hostPlatform.system; };
        in
        {
          go = goPkgs.go_1_27;
          buildGo127Module = goPkgs.buildGo127Module;
          mockgen = prev.mockgen.override { buildGoModule = final.buildGo127Module; };
          golangci-lint = goPkgs.callPackage "${go-tooling}/pkgs/by-name/go/golangci-lint/package.nix" {
            buildGo127Module = final.buildGo127Module;
          };
        };

      packages = forEachSupportedSystem ({ pkgs, system }:
        {
          speakeasy = pkgs.stdenv.mkDerivation {
            pname = "speakeasy";
            version = speakeasyVersion;

            src = pkgs.fetchurl {
              url = "https://github.com/speakeasy-api/speakeasy/releases/download/v${speakeasyVersion}/speakeasy_${speakeasyPlatforms.${system}}.zip";
              sha256 = speakeasyHashes.${system};
            };

            nativeBuildInputs = [ pkgs.unzip ];
            dontUnpack = true;

            installPhase = ''
              mkdir -p $out/bin
              unzip $src
              ls -al
              install -m755 speakeasy $out/bin/
            '';

            name = "speakeasy";
          };
        }
      );

      defaultPackage.x86_64-linux   = self.packages.x86_64-linux.speakeasy;
      defaultPackage.aarch64-linux  = self.packages.aarch64-linux.speakeasy;
      defaultPackage.x86_64-darwin  = self.packages.x86_64-darwin.speakeasy;
      defaultPackage.aarch64-darwin = self.packages.aarch64-darwin.speakeasy;

      devShells = forEachSupportedSystem ({ pkgs, system }:
        {
          default = pkgs.mkShell {
            GOTOOLCHAIN = "local";
            GOROOT = "${pkgs.go}/share/go";
            packages = with pkgs; [
              go
              gotools
              golangci-lint
              ginkgo
              (pkgs.callPackage "${goreleaser}/pkgs/goreleaser-pro" { })
              self.packages.${system}.speakeasy
              just
              mockgen
              jdk11
              yq-go
            ];
          };
        }
      );
    };
}
