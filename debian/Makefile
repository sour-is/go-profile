NAME=sour.is-profile
VERSION=$(shell cat VERSION)
RELEASE=$(shell cat RELEASE)

REPO_HOST="kapha.zt.sour.is"
REPO_PATH="/opt/web/pub/sour.is/debian/"

all: do-release do-build copy

clean:
	rm -r BUILD/*

do-release:
	export RELEASE=`cat RELEASE`; \
	echo `expr $${RELEASE} + 1` > RELEASE

do-build:
	export VERSION=`cat VERSION`; \
	export RELEASE=`cat RELEASE`; \
	export DATE=`date -u +%FT%TZ`; \
	export BUILD="BUILD/$(NAME)_$${VERSION}-$${RELEASE}"; \
	rm -rf ./$${BUILD} && \
	cp -r ROOT "$${BUILD}"; \
	sed -i.bak "s_Version:.*_Version: $${VERSION}-$${RELEASE}_" "$${BUILD}/DEBIAN/control"; \
	export GOOS=linux GOARCH=amd64; go build -o $${BUILD}/opt/sour.is/bin/profile \
                 -ldflags "-X main.AppVersion=$${VERSION}-$${RELEASE} -X main.AppBuild=$${DATE}"\
           sour.is/x/profile && \
	dpkg -b $${BUILD}

copy:
	export VERSION=`cat VERSION` && \
	export RELEASE=`cat RELEASE` && \
	export BUILD="BUILD/$(NAME)_$${VERSION}-$${RELEASE}" && \
	scp "$${BUILD}.deb" $(REPO_HOST):$(REPO_PATH) && \
	ssh $(REPO_HOST) -- $(REPO_PATH)scan.sh "$(NAME)_$${VERSION}-$${RELEASE}.deb";

deploy:
	ansible $(REPO_HOST) -b -m apt -a "name=sour.is-profile update_cache=yes state=latest"
