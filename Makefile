VERSION?=1.0.6
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
OCB?= ocb

.PHONY: all
all: config collector-bin collector-dist

.PHONY: config
config: artifacts/honeycomb-metrics-config.yaml

.PHONY: collector-bin
collector-bin: build/otelcol_hny_darwin_amd64 build/otelcol_hny_darwin_arm64 build/otelcol_hny_linux_amd64 build/otelcol_hny_linux_arm64 build/otelcol_hny_windows_amd64.exe

.PHONY: collector-dist
collector-dist: dist/otel-hny-collector_$(VERSION)_amd64.deb dist/otel-hny-collector_$(VERSION)_arm64.deb dist/otel-hny-collector_$(VERSION)_x86_64.rpm dist/otel-hny-collector_$(VERSION)_arm64.rpm 

.PHONY: release
release:
	$(MAKE) clean
	$(MAKE) test
	$(MAKE) artifacts/honeycomb-metrics-config.yaml
	$(MAKE) collector-bin
	$(MAKE) collector-dist
	cp build/otelcol_hny_* dist
	cp artifacts/honeycomb-metrics-config.yaml dist
	(cd dist && shasum -a 256 * > checksums.txt)

.PHONY: test
test: integration_test

.PHONY: integration_test
integration_test: test/test.sh build/otelcol_hny_$(GOOS)_$(GOARCH) artifacts/honeycomb-metrics-config.yaml
	./test/test.sh

# generate a configuration file for otel-collector that results in a favorable repackaging ratio
artifacts/honeycomb-metrics-config.yaml: config-generator.jq vendor-fixtures/hostmetrics-receiver-metadata.yaml
	mkdir -p ./artifacts
	yq -y -f ./config-generator.jq < ./vendor-fixtures/hostmetrics-receiver-metadata.yaml > ./artifacts/honeycomb-metrics-config.yaml

# copy hostmetrics metadata yaml file from the OpenTelemetry Collector repository, and prepend a note saying it's vendored
vendor-fixtures/hostmetrics-receiver-metadata.yaml:
	REMOTE_PATH='https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-contrib/141da3a5c4a1bf1570372e2890af383dd833167b/receiver/hostmetricsreceiver/metadata.yaml'; \
	curl $$REMOTE_PATH | sed "1s|^|# DO NOT EDIT! This file is vendored from $${REMOTE_PATH}"$$'\\\n\\\n|' > vendor-fixtures/hostmetrics-receiver-metadata.yaml

build/otelcol_hny_darwin_amd64:
	GOOS=darwin GOARCH=amd64 $(MAKE) build-binary-internal

build/otelcol_hny_darwin_arm64:
	GOOS=darwin GOARCH=arm64 $(MAKE) build-binary-internal

build/otelcol_hny_linux_amd64:
	GOOS=linux GOARCH=amd64 $(MAKE) build-binary-internal

build/otelcol_hny_linux_arm64:
	GOOS=linux GOARCH=arm64 $(MAKE) build-binary-internal

build/otelcol_hny_windows_amd64.exe:
	GOOS=windows GOARCH=amd64 EXTENSION=.exe $(MAKE) build-binary-internal

.PHONY: build-binary-internal
build-binary-internal: builder-config.yaml
	$(OCB) --output-path=build --config=builder-config.yaml $(OCB_OPTS)

dist/otel-hny-collector_%_amd64.deb: build/otelcol_hny_linux_amd64
	PACKAGE=deb ARCH=amd64 VERSION=$* $(MAKE) build-package-internal

dist/otel-hny-collector_%_arm64.deb: build/otelcol_hny_linux_arm64
	PACKAGE=deb ARCH=arm64 VERSION=$* $(MAKE) build-package-internal

dist/otel-hny-collector_%_x86_64.rpm: build/otelcol_hny_linux_amd64
	PACKAGE=rpm ARCH=amd64 VERSION=$* $(MAKE) build-package-internal

dist/otel-hny-collector_%_arm64.rpm: build/otelcol_hny_linux_arm64
	PACKAGE=rpm ARCH=arm64 VERSION=$* $(MAKE) build-package-internal

.PHONY: build-package-internal
build-package-internal:
	docker build -t otelcol-fpm packaging/fpm
	docker run --rm -v $(CURDIR):/repo -e VERSION=$(VERSION) -e ARCH=$(ARCH) -e PACKAGE=$(PACKAGE) otelcol-fpm

.PHONY: clean
clean:
	rm -f build/* compact-config.yaml test/tmp-* dist/* artifacts/*
