ROUTE_ASSET=internal/route/bindata_assetfs.go
ROUTE_FILES=public/*

SCHEMA_ASSET=cmd/profile/bindata.go
SCHEMA_FILES=schema/*

SOURCE=./cmd/*/*.go internal/*/*.go
CMD=sour.is/x/profile/cmd/profile
BINARY=bin/profile

all: $(BINARY)

clean:
	rm -f $(BINARY) $(SCHEMA_ASSET) $(ROUTE_ASSET)

fmt:   $(SOURCE) $(SCHEMA_ASSET) $(ROUTE_ASSET)
	find . -type f -name "*.go" -printf "%h\n"|sort -u|xargs go fmt
	go tool vet -composite=false

$(BINARY): $(SOURCE) $(SCHEMA_ASSET) $(ROUTE_ASSET)
	export DATE=`date -u +%FT%TZ`; \
	go build \
	   -o $(BINARY) \
	   -ldflags "-X main.AppVersion=VERSION-SNAPSHOT -X main.AppBuild=$${DATE}" \
	   $(CMD)

$(ROUTE_ASSET): $(ROUTE_FILES)
	export PATH=$$GOPATH/bin:$$PATH; go generate -v sour.is/x/profile/internal/route

$(SCHEMA_ASSET): $(SCHEMA_FILES)
	export PATH=$$GOPATH/bin:$$PATH; go-bindata -pkg main schema

deploy: clean $(SOURCE) $(SCHEMA_ASSET) $(ROUTE_ASSET)
	cd debian && make && make deploy

run: $(BINARY)
	bin/profile -vv serve

.PHONEY: clean deploy fmt run
