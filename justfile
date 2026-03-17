
build: build-go build-extension

reload: build
  go/build/release/chrest install jbcogiaaaaikinoljmplilmcnicpfoek
  chrest reload-extension

build-go:
  just go/build

build-extension:
  just extension/build

test: test-go test-mcp

test-go:
  just go/tests-go

test-mcp: build
  npx @modelcontextprotocol/inspector --cli --method tools/list go/build/release/chrest mcp
  npx @modelcontextprotocol/inspector --cli --method resources/list go/build/release/chrest mcp
  npx @modelcontextprotocol/inspector --cli --method resources/templates/list go/build/release/chrest mcp

dev-install-mcp: build
  go/build/release/chrest install-mcp

demo:
  vhs demo/demo.tape
