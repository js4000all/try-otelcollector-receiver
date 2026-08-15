.PHONY: build build-amd64 build-arm64 clean

DIST_NAME := ./dist/otelcol-practice
VERSION := 0.1.0
		
build: build-amd64 build-arm64

build-amd64:
	GOOS=linux GOARCH=amd64 ocb --config builder-config.yaml
	mv $(DIST_NAME) $(DIST_NAME)_$(VERSION)_linux_amd64

build-arm64:
	GOOS=linux GOARCH=arm64 ocb --config builder-config.yaml
	mv $(DIST_NAME) $(DIST_NAME)_$(VERSION)_linux_arm64

clean:
	rm -rf ./dist
