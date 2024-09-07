
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

.PHONY: build local_deploy sign

build: dist/options.html dist/options.js dist/main.js dist/chrest.zip;

FILE_SECRET_ISSUER := $(realpath secrets/jwt_issuer)
FILE_SECRET_SECRET := $(realpath secrets/jwt_secret)

sign: build
>pushd git rev-parse --show-toplevel
>git secret reveal $(FILE_SECRET_SECRET) $(FILE_SECRET_ISSUER)
>popd
>pushd dist/
>npx web-ext sign --channel=listed --amo-metadata=amo-metadata.json --api-key="$(shell cat $(FILE_SECRET_ISSUER))" --api-secret="$(shell cat $(FILE_SECRET_SECRET))"

local_deploy: build
>npx web-ext run -s dist/

dist/options.js: options.js
>mkdir -p $(dir $@)
>cp $< $@

dist/manifest.json: manifest.json
>mkdir -p $(dir $@)
>cp $< $@

dist/options.html: options.html
>mkdir -p $(dir $@)
>cp $< $@

dist/amo-metadata.json: amo-metadata.json
>mkdir -p $(dir $@)
>cp $< $@

dist/chrest_icon_128.png: chrest_icon_128.png
>mkdir -p $(dir $@)
>cp $< $@

files_js := $(shell find src -iname '*.js')

files_dist := dist/manifest.json dist/amo-metadata.json dist/chrest_icon_128.png

dist/main.js: $(files_js) $(files_dist)
>npm i
>mkdir -p $(dir $@)
>npx rollup --config
>npx web-ext lint -s dist/

dist/chrest.zip: dist/main.js dist/options.js dist/options.html
>zip -r "$@" dist/* -x "$@"


