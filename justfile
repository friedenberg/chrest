
build: build-go build-extension

reload: build
  go/build/release/chrest install jbcogiaaaaikinoljmplilmcnicpfoek
  chrest reload-extension

build-go:
  just go/build

build-extension:
  just extension/build

dev-install-mcp: build
  go/build/release/chrest install-mcp

demo:
  vhs demo/demo.tape
