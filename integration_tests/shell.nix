{ system ? builtins.currentSystem, pkgs ? import ../nix { inherit system; } }:
pkgs.mkShell {
  buildInputs = [
    pkgs.jq
    pkgs.go
    (import ../. { inherit pkgs; })
    pkgs.start-scripts
    pkgs.go-ethereum
    pkgs.pystarport
    pkgs.orchestrator
    pkgs.poetry
    pkgs.yarn
    pkgs.nodejs
    pkgs.git
    pkgs.dapp
    (pkgs.runCommand "solc" { } "mkdir -p $out/bin; ln -s ${pkgs.solc-static-versions.solc_0_6_11}/bin/solc-0.6.11 $out/bin/solc")
    (import ../nix/testenv.nix { inherit pkgs; })
  ];
}
