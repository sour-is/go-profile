ROUTE_ASSET=internal/route/bindata_assetfs.go
ROUTE_FILES=public/*

SCHEMA_ASSET=bindata.go
SCHEMA_FILES=schema/*

SOURCE=./*.go internal/*/*.go
BINARY=profile

all: $(BINARY)

clean:
	rm -f $(BINARY) $(SCHEMA_ASSET) $(ROUTE_ASSET)

$(BINARY): $(SOURCE) $(SCHEMA_ASSET) $(ROUTE_ASSET)
	go build

$(ROUTE_ASSET): $(ROUTE_FILES)
	export PATH=$$GOPATH/bin:$$PATH; cd internal/route; go-bindata-assetfs -pkg route -prefix ../../ ../../public/

$(SCHEMA_ASSET): $(SCHEMA_FILES)
	export PATH=$$GOPATH/bin:$$PATH; go-bindata -pkg main schema/

.PHONEY: clean
