# CMD :: the identity of what we're building: the executable name, image name, etc.
CMD=otelcol_hny

# Figure out the version of the project
CIRCLE_TAG?=$(shell git describe --tags --always) # compute a "tag" if not set by CI or a human
VERSION=$(CIRCLE_TAG:v%=%)
ifneq (,$(findstring -g,$(VERSION))) # if the version contains a git hash, it's a dev build
MAYBE_SNAPSHOT=--snapshot
else
MAYBE_SNAPSHOT=
endif

.PHONY: version
#: print out the detected version and other build info
version:
	@echo "CIRCLE_TAG: $(CIRCLE_TAG)"
	@echo "VERSION (build info & packaging): $(VERSION)"

# The Go platform info for the build host; cross-compile target are figured out differently
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

# Some tools needed for build.
YQ=bin/yq
OCB=bin/ocb
GORELEASER=bin/goreleaser

.PHONY: tools_exist
tools_exist: $(YQ) $(OCB) $(GORELEASER)

.PHONY: all
all: config image

.PHONY: config
config: artifacts/honeycomb-metrics-config.yaml

.PHONY: collector-bin
collector-bin: dist/otelcol_hny_darwin_amd64 dist/otelcol_hny_darwin_arm64 dist/otelcol_hny_linux_amd64 dist/otelcol_hny_linux_arm64 dist/otelcol_hny_windows_amd64.exe

.PHONY: collector-dist
collector-dist: dist/otel-hny-collector_$(VERSION)_amd64.deb dist/otel-hny-collector_$(VERSION)_arm64.deb dist/otel-hny-collector-$(VERSION)-x86_64.rpm dist/otel-hny-collector-$(VERSION)-arm64.rpm

.PHONY: release
release: artifacts/honeycomb-metrics-config.yaml $(GORELEASER)
	VERSION=$(VERSION) \
		$(GORELEASER) release $(MAYBE_SNAPSHOT) --clean

.PHONY: test
test: integration_test

.PHONY: integration_test
integration_test: test/test.sh artifacts/honeycomb-metrics-config.yaml build
	@echo "\n +++ Running integration tests\n"
	./test/test.sh

artifacts:
	mkdir -p artifacts

# generate a configuration file for otel-collector that results in a favorable repackaging ratio
artifacts/honeycomb-metrics-config.yaml: artifacts config-generator.jq vendor-fixtures/hostmetrics-receiver-metadata.yaml
	@echo "\n +++ Generating configuration for metrics rename and compaction\n"
	$(YQ) --yaml-output \
		--from-file ./config-generator.jq \
		< ./vendor-fixtures/hostmetrics-receiver-metadata.yaml \
		> ./artifacts/honeycomb-metrics-config.yaml

# copy hostmetrics metadata yaml file from the OpenTelemetry Collector repository, and prepend a note saying it's vendored
vendor-fixtures/hostmetrics-receiver-metadata.yaml:
	REMOTE_PATH='https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-contrib/141da3a5c4a1bf1570372e2890af383dd833167b/receiver/hostmetricsreceiver/metadata.yaml'; \
	curl $$REMOTE_PATH | sed "1s|^|# DO NOT EDIT! This file is vendored from $${REMOTE_PATH}"$$'\\\n\\\n|' > vendor-fixtures/hostmetrics-receiver-metadata.yaml

#
# Source Generation
#

SRC_DIR=cmd/$(CMD)
$(SRC_DIR):
	mkdir -p $@

GO_SOURCES := $(SRC_DIR)/go.mod $(SRC_DIR)/go.sum $(wildcard $(SRC_DIR)/*.go)
.PHONY: source
#: generate the source for the custom build
source: $(GO_SOURCES)
# "&:" here is a grouped target where all the files listed in GO_SOURCES are generated *once* by this rule
$(GO_SOURCES) &: $(SRC_DIR) $(OCB) builder-config.yaml
	@echo "\n +++ Generating $(CMD) sources\n"
	$(OCB) --output-path=$(SRC_DIR) --skip-compilation --name=$(CMD) --version=$(VERSION) --config=builder-config.yaml


.PHONY: build
#: build binary for the current platform
build: $(GO_SOURCES) $(GORELEASER)
	@echo "\n +++ Building $@\n"
	VERSION=$(VERSION) \
		$(GORELEASER) build $(MAYBE_SNAPSHOT) --clean --single-target

.PHONY: build_all
#: build binaries for all target platforms
build_all: $(GO_SOURCES) $(GORELEASER)
	@echo "\n +++ Building $@\n"
	VERSION=$(VERSION) \
		$(GORELEASER) build $(MAYBE_SNAPSHOT) --clean

.PHONY: package
#: build RPM and DEB packages
package: $(GO_SOURCES) $(GORELEASER)
	@echo "\n +++ Packaging \n"
	VERSION=$(VERSION) \
		$(GORELEASER) release $(MAYBE_SNAPSHOT) --clean --skip archive,ko,publish

.PHONY: image
KO_DOCKER_REPO ?= ko.local
#: build a docker image; set KO_DOCKER_REPO to push to a registry
image: $(GO_SOURCES) $(GORELEASER)
	@echo "\n +++ Building image \n"
	VERSION=$(VERSION) \
		$(GORELEASER) release $(MAYBE_SNAPSHOT) --clean --skip archive,nfpm,publish

.PHONY: clean
clean:
	@echo "\n +++ Cleaning up generated things\n"
	rm -rf cmd/otelcol_hny build/* compact-config.yaml test/tmp-* dist/* artifacts/*

.PHONY: squeaky_clean
squeaky_clean: clean
	git clean --force -X bin/

OTELCOL_VERSION=$(shell $(YQ) --raw-output ".dist.otelcol_version" < builder-config.yaml)
#: symlink for convenience to OpenTelemetry Collector Builder for this project
$(OCB): $(OCB)-$(OTELCOL_VERSION)
	ln -s -f $(shell basename $<) $@

#: the OpenTelemetry Collector Builder for this project; to be downloaded when not present
$(OCB)-$(OTELCOL_VERSION):
	@echo "\n +++ Retrieve required OpenTelemetry Collector Builder v$(OTELCOL_VERSION)\n"
	curl --fail --location --output $@ \
		"https://github.com/open-telemetry/opentelemetry-collector/releases/download/cmd/builder/v${OTELCOL_VERSION}/ocb_${OTELCOL_VERSION}_${GOOS}_${GOARCH}"
	chmod u+x $@

GORELEASER_VERSION ?= $(shell cat .tool-versions | grep goreleaser | cut -d' ' -f 2)
# ensure the dockerize command is available
$(GORELEASER): $(GORELEASER)_$(GORELEASER_VERSION).tar.gz
	tar xzvmf $< -C bin goreleaser
	chmod u+x $@

ifeq (aarch64, $(shell uname -m))
GORELEASER_RELEASE_ASSET = goreleaser_$(shell uname -s)_arm64.tar.gz
else
GORELEASER_RELEASE_ASSET = goreleaser_$(shell uname -s)_$(shell uname -m).tar.gz
endif
$(GORELEASER)_$(GORELEASER_VERSION).tar.gz:
	@echo
	@echo "+++ Retrieving goreleaser tool for build and releasing."
	@echo
# make sure that file is available
ifeq (, $(shell command -v file))
	sudo apt-get update
	sudo apt-get -y install file
endif
	curl --location --silent --show-error \
		--output goreleaser_tmp.tar.gz \
		https://github.com/goreleaser/goreleaser/releases/download/v$(GORELEASER_VERSION)/$(GORELEASER_RELEASE_ASSET) \
	&& file goreleaser_tmp.tar.gz | grep --silent gzip \
	&& mv goreleaser_tmp.tar.gz $@ || (echo "Failed to download goreleaser. Got:"; cat goreleaser_tmp.tar.gz ; echo "" ; exit 1)

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
