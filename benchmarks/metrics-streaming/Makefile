include ../../.bingo/Variables.mk

.PHONY: help
help: ## Display this help and any documented user-facing targets. Other undocumented targets may be present in the Makefile.
help:
	@awk 'BEGIN {FS = ": ##"; printf "Usage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_\.\-\/%]+: ##/ { printf "  %-45s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: proto
proto: ## Regenerate Go from proto.
proto: $(BUF)
	@echo ">> regenerating Prometheus and OpenTelemetry protocols"
	@$(BUF) generate
	# For some reasons buf generates this unused import, kill it manually for now and reformat.
	@find io/prometheus/write/ -type f -exec sed -i.bak 's/_ "github.com\/gogo\/protobuf\/gogoproto"//g' {} \;
	@find io/prometheus/write/ -name "*.bak" -delete
	@go fmt ./io/...
