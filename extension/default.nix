{
  lib,
  mkBunDerivation,
  fetchBunDeps,
  jq,
  zip,
  browserType,
}:

let
  src = lib.fileset.toSource {
    root = ./.;
    fileset = lib.fileset.unions [
      ./src
      ./assets
      ./manifest-common.json
      ./manifest-chrome.json
      ./manifest-firefox.json
      ./rolldown.config.mjs
      ./package.json
      ./bun.lock
      ./bun.nix
      ./zz-firefox-amo-metadata.json
    ];
  };
in
mkBunDerivation {
  pname = "chrest-extension-${browserType}";
  version = "1.16.0";
  inherit src;
  packageJson = ./package.json;
  bunDeps = fetchBunDeps {
    bunNix = ./bun.nix;
  };

  nativeBuildInputs = [
    jq
    zip
  ];

  buildPhase = ''
    runHook preBuild

    mkdir -p dist-${browserType}

    jq -s 'reduce .[] as $i ({}; . + $i)' \
      manifest-common.json manifest-${browserType}.json \
      > dist-${browserType}/manifest.json

    cp src/* assets/* dist-${browserType}/

    BROWSER_TYPE=${browserType} bun run build

    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall

    mkdir -p "$out"
    cp -r dist-${browserType} "$out/"
    ${lib.optionalString (browserType == "firefox") ''
      cp zz-firefox-amo-metadata.json "$out/dist-${browserType}/"
    ''}

    # Info-ZIP's zip reads mtime from stat() and does not honor
    # SOURCE_DATE_EPOCH. Normalize mtimes to the earliest representable
    # DOS date (1980-01-01 UTC) so two builds produce identical zips.
    find "$out/dist-${browserType}" -exec touch -h -t 198001010000 {} +

    ( cd "$out" && \
      find dist-${browserType} -type f -print0 | LC_ALL=C sort -z | \
        TZ=UTC xargs -0 zip -X -q dist-${browserType}.zip )

    runHook postInstall
  '';
}
