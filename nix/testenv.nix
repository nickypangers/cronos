{ pkgs }:
pkgs.poetry2nix.mkPoetryEnv {
  projectDir = ../integration_tests;
  overrides = pkgs.poetry2nix.overrides.withDefaults (self: super: {
    cprotobuf = super.cprotobuf.overridePythonAttrs (
      old: {
        buildInputs = (old.buildInputs or [ ]) ++ [ self.cython ];
      }
    );
  });
}
