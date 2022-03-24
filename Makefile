MAKEFLAGS += --silent

BIN_DIR=/opt/scanpi
BIN=scanpi
BUILD_DIR=build
RELEASE_DIR=$(BUILD_DIR)/release
TMP_DIR=$(BUILD_DIR)/tmp
VERSION := $(shell cat VERSION)
PLATFORM := $(shell uname -m)

ARCH :=
	ifeq ($(PLATFORM),x86_64)
		ARCH = amd64
	endif
	ifeq ($(PLATFORM),aarch64)
		ARCH = arm64
	endif
	ifeq ($(PLATFORM),armv7l)
		ARCH = armhf
	endif
GOARCH :=
	ifeq ($(ARCH),amd64)
		GOARCH = amd64
	endif
	ifeq ($(ARCH),arm64)
		GOARCH = arm64
	endif
	ifeq ($(ARCH),armhf)
		GOARCH = arm
	endif

package: clean prepare cp compile control
	@echo Building package...
	chmod --quiet 0555 $(TMP_DIR)/DEBIAN/p* || true
	fakeroot dpkg-deb -b -z9 $(TMP_DIR) $(RELEASE_DIR)

clean:
	rm -rf $(TMP_DIR) $(RELEASE_DIR)

prepare:
	@echo Prepare...
	mkdir -p $(TMP_DIR)/$(BIN_DIR) $(RELEASE_DIR)

cp:
	cp -R deb/* $(TMP_DIR)

compile:
	go mod tidy
	go mod vendor > /dev/null 2>&1
	GOOS=linux GOARCH=$(GOARCH) go build -o $(TMP_DIR)/$(BIN_DIR)/$(BIN) main.go

control:
	$(eval size=$(shell du -sbk $(TMP_DIR)/ | grep -o '[0-9]*'))
	@sed -i "s/==version==/$(VERSION)/g;s/==size==/$(size)/g;s/==architecture==/$(ARCH)/g" "$(TMP_DIR)/DEBIAN/control"

test:
	go test ./... -race -cover
