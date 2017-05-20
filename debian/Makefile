NAME=sour.is-profile
VERSION=$(shell cat VERSION)
RELEASE=$(shell cat RELEASE)

REPO_IP="172.22.141.171"
REPO_PATH="/opt/web/pub/sour.is/debian/"

all: release build copy

clean:
	rm -r BUILD/*

release:
	export RELEASE=`cat RELEASE`; \
	echo `expr $${RELEASE} + 1` > RELEASE

build:
	export VERSION=`cat VERSION`; \
	export RELEASE=`cat RELEASE`; \
	export DATE=`date -u +%FT%TZ`; \
	export BUILD="BUILD/$(NAME)_$${VERSION}-$${RELEASE}"; \
	rm -rf ./$${BUILD}; \
	cp -r ROOT "$${BUILD}"; \
	export SED="s_Version:.*_Version: $${VERSION}-$${RELEASE}_"; \
	sed -i "$$SED" "$${BUILD}/DEBIAN/control"; \
	go build -o $${BUILD}/opt/sour.is/bin/profile \
                 -ldflags "-X main.APP_VERSION=$${VERSION}-$${RELEASE} -X main.APP_BUILD=$${DATE}"\
           sour.is/x/profile; \
	dpkg -b $${BUILD};

copy:
	export VERSION=`cat VERSION`; \
	export RELEASE=`cat RELEASE`; \
	export BUILD="BUILD/$(NAME)_$${VERSION}-$${RELEASE}"; \
	scp "$${BUILD}.deb" $(REPO_IP):$(REPO_PATH); \
	ssh $(REPO_IP) -- $(REPO_PATH)scan.sh "$(REPO_PATH)$(NAME)_$${VERSION}-$${RELEASE}.deb";

deploy:
	ansible kapha -s -m apt -a "name=sour.is-profile update_cache=yes state=latest"
