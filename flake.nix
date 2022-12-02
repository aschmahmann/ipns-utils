{
  description = "";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, flake-utils, nixpkgs }:
    flake-utils.lib.eachDefaultSystem (system: 
      let
          pkgs = import nixpkgs {
            inherit system;
          };
          ipns-utils = pkgs.buildGoModule {
            name = "ipns-utils";
            src = ./.;
            vendorSha256 = "sha256-HndfSJmDnsYExXSdbNLMjuJBYu8eQiEhpC+MyuNP2ks=";
          };
      in rec {
        packages.default = ipns-utils;
      }
    );
}
