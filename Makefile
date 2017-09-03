ROUTE_ASSET=internal/route/bindata_assetfs.go
ROUTE_FILES=public/*

SCHEMA_ASSET=bindata.go
SCHEMA_FILES=schema/*

SOURCE=./*.go internal/*/*.go
BINARY=profile

all: $(BINARY)

clean:
	rm -f $(BINARY) $(SCHEMA_ASSET) $(ROUTE_ASSET)

fmt:   $(SOURCE) $(SCHEMA_ASSET) $(ROUTE_ASSET)
	find . -type f -name "*.go" -printf "%h\n"|sort -u|xargs go fmt
	go tool vet -composite=false

$(BINARY): $(SOURCE) $(SCHEMA_ASSET) $(ROUTE_ASSET)
	go build

$(ROUTE_ASSET): $(ROUTE_FILES)
	export PATH=$$GOPATH/bin:$$PATH; cd internal/route; go-bindata-assetfs -pkg route -prefix ../../ ../../public/ ../../public/ui

$(SCHEMA_ASSET): $(SCHEMA_FILES)
	export PATH=$$GOPATH/bin:$$PATH; go-bindata -pkg main schema/

deploy: clean $(SOURCE) $(SCHEMA_ASSET) $(ROUTE_ASSET)
	cd debian && make && make deploy

run: $(BINARY)
	./profile -vv serve

.PHONEY: clean deploy fmt run
