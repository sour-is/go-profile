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
	export DATE=`date -u +%FT%TZ`; \
	go build -ldflags "-X main.AppVersion=VERSION-SNAPSHOT -X main.AppBuild=$${DATE}"

$(ROUTE_ASSET): $(ROUTE_FILES)
	go generate -v sour.is/x/profile/internal/route

$(SCHEMA_ASSET): $(SCHEMA_FILES)
	go run github.com/sour-is/go-assetfs/cmd/assetfs -pkg main schema/

deploy: clean $(SOURCE) $(SCHEMA_ASSET) $(ROUTE_ASSET)
	cd debian && make && make deploy

run: $(BINARY)
	./profile -vv serve

.PHONEY: clean deploy fmt run
