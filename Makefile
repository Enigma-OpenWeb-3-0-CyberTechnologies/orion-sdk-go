# Allow setting different go version from the command line. E.g.,`make GO=go1.19.4 binary`
GO       ?= go
TIMEOUT  = 20m
PKGS     = $(or $(PKG),$(shell env GO111MODULE=on $(GO) list ./...))
TESTPKGS = $(shell env GO111MODULE=on $(GO) list -f \
		   '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' \
		   $(PKGS))

COVERAGE_MODE    = atomic
COVERAGE_PROFILE = $(COVERAGE_DIR)/profile.out
COVERAGE_XML     = $(COVERAGE_DIR)/coverage.xml
COVERAGE_HTML    = $(COVERAGE_DIR)/index.html
BIN = $(CURDIR)/bin

$(BIN):
	@mkdir -p $@

$(BIN)/%: | $(BIN)
	@tmp=$$(mktemp -d); \
		env GO11MODULE=off GOPATH=$$tmpp GOBIN=$(BIN) $(GO) get $(PACKAGE) \
		|| ret=$$?;
	rm -rf $$tmp ; exit $$ret

$(BIN)/golangci-lint: PACKAGE=github.com/golangci/golangci-lint/cmd/golangci-lint

GOLINT = $(BIN)/golangci-lint

lint: | $(GOLINT)
	$(GOLINT) run  

GOCOV = $(BIN)/gocov
$(BIN)/gocov: PACKAGE=github.com/axw/gocov/...

GOCOVXML = $(BIN)/gocov-xml
$(BIN)/gocov-xml: PACKAGE=github.com/AlekSi/gocov-xml

GO2XUNIT = $(BIN)/go2xunit
$(BIN)/go2xunit: PACKAGE=github.com/tebeka/go2xunit

.PHONY: fmt
fmt:
	$(GO) fmt $(PKGS)

.PHONY: goimports
goimports:
	find . -name \*.go -not -path "./pkg/types/*" -exec goimports -w -l {} \;

.PHONY: binary
binary:
	$(GO) build -o $(BIN)/bdb github.com/hyperledger-labs/orion-server/cmd/bdb

.PHONY: test
test-script:
	scripts/run-unit-tests.sh

.PHONY: clean
clean: 
	@rm -rf $(BIN)
	@rm -rf test/tests.* test/coverage.*

.PHONY: protos
protos: 
	docker run -it -v `pwd`:`pwd` -w `pwd` sykesm/fabric-protos:0.2 scripts/compile_go_protos.sh

TEST_TARGETS := test-default test-bench test-short test-verbose test-race
test-bench:   ARGS=-run=__absolutelynothing__ -bench=.
test-short:   ARGS=-short
test-verbose: ARGS=-v
test-race:    ARGS=-race
$(TEST_TARGETS): test
check test tests:
	$(GO) build -o $(BIN)/bdb github.com/hyperledger-labs/orion-server/cmd/bdb
	$(GO) test -timeout $(TIMEOUT) $(ARGS) $(TESTPKGS)

test-coverage-tools: | $(GOCOVMERGE) $(GOCOV) $(GOCOVXML) 
test-coverage: COVERAGE_DIR := $(CURDIR)/test/coverage.$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
test-coverage: test-coverage-tools
	mkdir -p $(COVERAGE_DIR)/coverage
	$(GO) test \
		-coverpkg=$$($(GO) list -f '{{ join .Deps "\n" }}' $(TESTPKGS) | \
		grep '^$(MODULE)/' | \
		tr '\n' ',' | sed 's/,$$//') \
		-covermode=$(COVERAGE_MODE) \
		-coverprofile="$(COVERAGE_PROFILE)" $(TESTPKGS)
	$(GO) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	$(GOCOV) convert $(COVERAGE_PROFILE) | $(GOCOVXML) > $(COVERAGE_XML)
