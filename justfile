
build: build-go build-extension

reload: build
  pushd go/
  ./build/chrest install jbcogiaaaaikinoljmplilmcnicpfoek
  popd
  chrest reload-extension

build-go:
  pushd go/
  just go/

build-extension:
  pushd extension
  just extension/
