VERSION?=$(shell git describe --tags --always)
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
YQ=bin/yq
OCB=bin/ocb

.PHONY: all
all: config collector-bin collector-dist

.PHONY: config
config: artifacts/honeycomb-metrics-config.yaml

.PHONY: collector-bin
collector-bin: build/otelcol_hny_darwin_amd64 build/otelcol_hny_darwin_arm64 build/otelcol_hny_linux_amd64 build/otelcol_hny_linux_arm64 build/otelcol_hny_windows_amd64.exe

.PHONY: collector-dist
collector-dist: dist/otel-hny-collector_$(VERSION)_amd64.deb dist/otel-hny-collector_$(VERSION)_arm64.deb dist/otel-hny-collector-$(VERSION)-x86_64.rpm dist/otel-hny-collector-$(VERSION)-arm64.rpm

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

artifacts:
	mkdir -p artifacts

# generate a configuration file for otel-collector that results in a favorable repackaging ratio
artifacts/honeycomb-metrics-config.yaml: artifacts config-generator.jq vendor-fixtures/hostmetrics-receiver-metadata.yaml
	$(YQ) --yaml-output \
		--from-file ./config-generator.jq \
		< ./vendor-fixtures/hostmetrics-receiver-metadata.yaml \
		> ./artifacts/honeycomb-metrics-config.yaml

# copy hostmetrics metadata yaml file from the OpenTelemetry Collector repository, and prepend a note saying it's vendored
vendor-fixtures/hostmetrics-receiver-metadata.yaml:
	REMOTE_PATH='https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-contrib/141da3a5c4a1bf1570372e2890af383dd833167b/receiver/hostmetricsreceiver/metadata.yaml'; \
	curl $$REMOTE_PATH | sed "1s|^|# DO NOT EDIT! This file is vendored from $${REMOTE_PATH}"$$'\\\n\\\n|' > vendor-fixtures/hostmetrics-receiver-metadata.yaml

src: $(OCB) builder-config.yaml
	$(OCB) --output-path=src --skip-compilation --name=otelcol_hny --version=$(VERSION) --config=builder-config.yaml

.PHONY: build
#: build the Honeycomb OpenTelemetry Collector for the current host's platform
build: build/otelcol_hny_$(GOOS)_$(GOARCH)

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
build-binary-internal: src builder-config.yaml
	CGO_ENABLED=0 go build -C ./src -o ../build/otelcol_hny_$(GOOS)_$(GOARCH)$(EXTENSION) ./...

dist/otel-hny-collector_%_amd64.deb: build/otelcol_hny_linux_amd64
	PACKAGE=deb ARCH=amd64 VERSION=$* $(MAKE) build-package-internal

dist/otel-hny-collector_%_arm64.deb: build/otelcol_hny_linux_arm64
	PACKAGE=deb ARCH=arm64 VERSION=$* $(MAKE) build-package-internal

dist/otel-hny-collector-%-x86_64.rpm: build/otelcol_hny_linux_amd64
	PACKAGE=rpm ARCH=amd64 VERSION=$* $(MAKE) build-package-internal

dist/otel-hny-collector-%-arm64.rpm: build/otelcol_hny_linux_arm64
	PACKAGE=rpm ARCH=arm64 VERSION=$* $(MAKE) build-package-internal

.PHONY: build-package-internal
build-package-internal:
	docker build -t otelcol-fpm packaging/fpm
	docker run --rm -v $(CURDIR):/repo -e VERSION=$(VERSION) -e ARCH=$(ARCH) -e PACKAGE=$(PACKAGE) otelcol-fpm

.PHONY: clean
clean:
	rm -rf src build/* compact-config.yaml test/tmp-* dist/* artifacts/*

OTELCOL_VERSION=$(shell $(YQ) --raw-output ".dist.otelcol_version" < builder-config.yaml)
#: symlink for convenience to OpenTelemetry Collector Builder for this project
$(OCB): $(OCB)-$(OTELCOL_VERSION)
	ln -s -f $(shell basename $<) $@

#: the OpenTelemetry Collector Builder for this project; to be downloaded when not present
$(OCB)-$(OTELCOL_VERSION):
	curl --fail --location --output $@ \
		"https://github.com/open-telemetry/opentelemetry-collector/releases/download/cmd/builder/v${OTELCOL_VERSION}/ocb_${OTELCOL_VERSION}_${GOOS}_${GOARCH}"
	chmod u+x $@

JOB ?= build
#: run a CI job in docker locally, set JOB=some-job to override default 'build'
ci_local_exec: local_ci_prereqs
	circleci local execute $(JOB) --config .circleci/config-processed.yml

### Utilities

# To use the circleci CLI to run jobs on your laptop.
circle_cli_docs_url = https://circleci.com/docs/local-cli/
local_ci_prereqs: forbidden_in_real_ci circle_cli_available .circleci/config-processed.yml

# the config must be processed to do things like expand matrix jobs.
.circleci/config-processed.yml: circle_cli_available .circleci/config.yml
	circleci config process .circleci/config.yml > .circleci/config-processed.yml

circle_cli_available:
ifneq (, $(shell which circleci))
	@echo "🔎:✅ circleci CLI available"
else
	@echo "🔎:💥 circleci CLI command not available for local run."
	@echo ""
	@echo "   ❓ Is it installed? For more info: ${circle_cli_docs_url}\n\n" && exit 1
endif

forbidden_in_real_ci:
ifeq ($(CIRCLECI),) # if not set, safe to assume not running in CircleCI compute
	@echo "🔎:✅ not running in real CI"
else
	@echo "🔎:🛑 CIRCLECI environment variable is present, a sign that we're running in real CircleCI compute."
	@echo ""
	@echo "   🙈 circleci CLI can't local execute in Circle. That'd be 🍌🍌🍌."
	@echo "" && exit 1
endif
