
SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --output-sync=target
# n_prc := $(shell sysctl -n hw.logicalcpu)
# MAKEFLAGS := --jobs=$(n_prc)
timeout := 10
cmd_bats := BATS_TEST_TIMEOUT=$(timeout) bats --tap

ifeq ($(origin .RECIPEPREFIX), undefined)
				$(error This Make does not support .RECIPEPREFIX. Please use GNU Make 4.0 or later)
endif
.RECIPEPREFIX = >

.PHONY: build local_deploy sign-firefox

build: dist/firefox.zip dist/chrome.zip;

FILE_SECRET_ISSUER := $(realpath secrets/jwt_issuer)
FILE_SECRET_SECRET := $(realpath secrets/jwt_secret)

sign-firefox: dist-firefox.zip
>pushd dist-firefox/
>dir_art=dist-firefox-artifacts
>mkdir -p "$$dir_art"
>npx web-ext sign --artifacts-dir "$$dir_art" --channel=listed --amo-metadata=amo-metadata.json --api-key="$(shell cat $(FILE_SECRET_ISSUER))" --api-secret="$(shell cat $(FILE_SECRET_SECRET))"

files_js_copy_to_dist := $(shell find common/src -type f)

dist/firefox.zip: dist/firefox/manifest.json dist/firefox/main.js $(shell find firefox common -type f)
>mkdir -p $(dir $@)
>dir="$(@:.zip=)"
>find common -maxdepth 1 -type f ! -iname "*manifest.json" -exec cp {} "$$dir" \;
>find firefox -maxdepth 1 -type f ! -iname "*manifest.json" -exec cp {} "$$dir" \;
>npx web-ext lint -s "$$dir"
>zip -r "$@" $$dir/*

files_js_copy_to_dist_firefox := $(shell find firefox/src -type f)

dist/firefox/main.js: firefox/src/browser.js $(files_js_copy_to_dist_firefox) $(files_js_copy_to_dist)
>src="$(dir $@)/src"
>mkdir -p "$$src"
>cp $^ "$$src"
>pushd "$(dir $@)"
>npx rollup --config ../../common/rollup.config.mjs > "$(abspath $@)"

dist/firefox/manifest.json: common/manifest.json firefox/manifest.json
>mkdir -p $(dir $@)
>cat $^ | jq -s 'reduce .[] as $$item ({}; . + $$item)' > "$@"

dist/chrome.zip: dist/chrome/manifest.json dist/chrome/main.js $(shell find chrome common -type f)
>mkdir -p $(dir $@)
>dir="$(@:.zip=)"
>find common -maxdepth 1 -type f ! -iname "*manifest.json" -exec cp {} "$$dir" \;
>find chrome -maxdepth 1 -type f ! -iname "*manifest.json" -exec cp {} "$$dir" \;
>zip -r "$@" $$dir/*

files_js_copy_to_dist_chrome := $(shell find chrome/src -type f)

dist/chrome/main.js: chrome/src/browser.js $(files_js_copy_to_dist_chrome) $(files_js_copy_to_dist)
>src="$(dir $@)/src"
>mkdir -p "$$src"
>cp $^ "$$src"
>pushd "$(dir $@)"
>npx rollup --config ../../common/rollup.config.mjs > "$(abspath $@)"

dist/chrome/manifest.json: common/manifest.json chrome/manifest.json
>mkdir -p $(dir $@)
>cat $^ | jq -s 'reduce .[] as $$item ({}; . + $$item)' > "$@"

local_deploy: build
>npx web-ext run -s dist/

