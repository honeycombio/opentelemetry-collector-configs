all: collector
collector: build/otelcol-hny

build/otelcol-hny: builder-config.yaml
	opentelemetry-collector-builder --output-path=build --name=hny-otel --config=builder-config.yaml

clean:
	rm build/*

.PHONY: all collector clean