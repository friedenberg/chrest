
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

build: dist-firefox.zip;

FILE_SECRET_ISSUER := $(realpath secrets/jwt_issuer)
FILE_SECRET_SECRET := $(realpath secrets/jwt_secret)

sign-firefox: dist-firefox.zip
>pushd dist-firefox/
>dir_art=dist-firefox-artifacts
>mkdir -p "$$dir_art"
>npx web-ext sign --artifacts-dir "$$dir_art" --channel=listed --amo-metadata=amo-metadata.json --api-key="$(shell cat $(FILE_SECRET_ISSUER))" --api-secret="$(shell cat $(FILE_SECRET_SECRET))"

files_copy_to_dist := chrest_icon_128.png options.js options.html
files_copy_to_dist_firefox := amo-metadata.json manifest.json
files_js_copy_to_dist := $(shell find src -iname '*.js')

dist-firefox.zip: dist-firefox/main.js $(addprefix dist-firefox/,$(files_copy_to_dist_firefox) $(files_copy_to_dist))
>dir="$(@:.zip=)"
>npx web-ext lint -s "$$dir"
>zip -r "$@" $$dir/*

dist-firefox/%: %
>mkdir -p $(dir $@)
>cp $< $@

dist-firefox/main.js: $(files_js_copy_to_dist) # TODO add firefox config file
>mkdir -p $(dir $@)
>npx rollup --config > "$@"

dist-firefox/manifest.json: manifest.json manifest-firefox.json
>mkdir -p $(dir $@)
>cat $^ | jq -s 'reduce .[] as $$item ({}; . + $$item)' > "$@"

local_deploy: build
>npx web-ext run -s dist/

