.DEFAULT_GOAL := help

BENCH_PKG  ?= ./internal/engine/
BENCH      ?= .
BENCHTIME  ?= 1s
PROFDIR    ?= profiles

.PHONY: help
help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the codesearch binary
	go build -o codesearch .

.PHONY: run
run: ## Show CLI usage (pass args with: go run . search foo)
	go run .

.PHONY: test
test: ## Run tests with the race detector
	go test -race ./...

.PHONY: fmt
fmt: ## Format all Go source
	gofmt -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: check
check: fmt vet test ## Format, vet, and test

.PHONY: bench
bench: ## Run benchmarks (BENCH=Search to filter, BENCHTIME=5s to lengthen)
	go test $(BENCH_PKG) -run '^$$' -bench '$(BENCH)' -benchmem -benchtime $(BENCHTIME)

.PHONY: bench-save
bench-save: ## Save a benchmark baseline to bench-base.txt
	go test $(BENCH_PKG) -run '^$$' -bench '$(BENCH)' -benchmem -benchtime $(BENCHTIME) -count 6 \
		| tee bench-base.txt

.PHONY: bench-compare
bench-compare: ## Compare current benchmarks against bench-base.txt (needs benchstat)
	@command -v benchstat >/dev/null 2>&1 || { \
		echo "benchstat not found. Install it with:"; \
		echo "  go install golang.org/x/perf/cmd/benchstat@latest"; \
		exit 1; }
	@test -f bench-base.txt || { echo "No bench-base.txt. Run 'make bench-save' first."; exit 1; }
	go test $(BENCH_PKG) -run '^$$' -bench '$(BENCH)' -benchmem -benchtime $(BENCHTIME) -count 6 \
		| tee bench-new.txt
	benchstat bench-base.txt bench-new.txt

.PHONY: profile
profile: ## Record CPU and memory profiles into $(PROFDIR)/
	@mkdir -p $(PROFDIR)
	go test $(BENCH_PKG) -run '^$$' -bench '$(BENCH)' -benchtime $(BENCHTIME) \
		-cpuprofile $(PROFDIR)/cpu.prof -memprofile $(PROFDIR)/mem.prof
	@echo
	@echo "Inspect with:"
	@echo "  go tool pprof -top -nodecount=15 $(PROFDIR)/cpu.prof"
	@echo "  go tool pprof -top -nodecount=15 -sample_index=alloc_space $(PROFDIR)/mem.prof"
	@echo "  go tool pprof -http=:8080 $(PROFDIR)/cpu.prof"

.PHONY: clean
clean: ## Remove build, benchmark, and profile artifacts
	rm -f codesearch bench-base.txt bench-new.txt
	rm -rf $(PROFDIR)
