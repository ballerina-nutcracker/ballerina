# Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
#
# WSO2 LLC. licenses this file to you under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SHELL := /bin/bash
PYTHON ?= python3
LIST_WORKSPACE_MODULES = go list -m -f '{{if .Main}}{{.Dir}}{{end}}' all
WORKSPACE_MODULE_DIRS := $(shell $(LIST_WORKSPACE_MODULES))
WORKSPACE_MODULES := . $(patsubst $(CURDIR)/%,%,$(filter-out $(CURDIR),$(WORKSPACE_MODULE_DIRS)))
BUILD_MODULES := $(filter-out compiler-tools/%,$(WORKSPACE_MODULES))
LINT_MODULES := $(filter-out compiler-tools/%,$(WORKSPACE_MODULES))
BUILD_MODULE_TARGETS := $(addprefix build-module/,$(BUILD_MODULES))
VET_MODULE_TARGETS := $(addprefix vet-module/,$(WORKSPACE_MODULES))
LINT_MODULE_TARGETS := $(addprefix lint-module/,$(LINT_MODULES))
TEST_RUNNER = .github/scripts/run_native_tests.py

.PHONY: build vet lint force test test-coverage test-race check update-testdata release install-hooks \
	test-wasm test-wasm-corpus-light test-wasm-corpus-integration benchmark-corpus

force:

install-hooks:
	@git config --local core.hooksPath .githooks
	@echo "Git hooks installed."

release:
	@bash .github/scripts/release_dist.sh "$(VERSION)" "$(or $(REMOTE),origin)"

build: $(BUILD_MODULE_TARGETS)

build-module/%: force
	@echo "Building $*"
	@(cd "$*" && go build ./...)

test:
	@$(PYTHON) $(TEST_RUNNER)

test-coverage:
	@$(PYTHON) $(TEST_RUNNER) --with-coverage

test-race:
	@$(PYTHON) $(TEST_RUNNER) --race

vet: $(VET_MODULE_TARGETS)

vet-module/%: force
	@echo "Vetting $*"
	@(cd "$*" && go vet ./...)

lint: $(LINT_MODULE_TARGETS)

lint-module/%: force
	@config="$(CURDIR)/.golangci.yml"; \
	if [[ -f "$(CURDIR)/$*/.golangci.yml" ]]; then \
		config="$(CURDIR)/$*/.golangci.yml"; \
	fi; \
	echo "Linting $* ($$config)"; \
	(cd "$*" && golangci-lint run --allow-parallel-runners --concurrency 1 --config "$$config" ./...)

check: build vet lint test

update-testdata:
	@set -uo pipefail; \
	while IFS= read -r dir; do \
		case "$$dir" in */compiler-tools/*) continue ;; esac; \
		(cd "$$dir" && go test ./... -update) || true; \
	done < <($(LIST_WORKSPACE_MODULES))

test-wasm:
	@set -euo pipefail; \
	wasm_exec="$$(go env GOROOT)/lib/wasm/go_js_wasm_exec"; \
	while IFS= read -r dir; do \
		case "$$dir" in */compiler-tools/*) continue ;; esac; \
		packages="$$(cd "$$dir" && go list ./... | sed -e '/^github.com\/ballerina-nutcracker\/ballerina\/corpus$$/d' -e '/^github.com\/ballerina-nutcracker\/ballerina\/cli\/internal\/nativerunner$$/d')"; \
		if [[ -n "$$packages" ]]; then \
			go test -p=1 -skip 'TestParseCorpusFiles|TestJBalUnitTests|TestJBalUnitBIRTests' -timeout 30m $$packages -exec="$$wasm_exec"; \
		fi; \
	done < <($(LIST_WORKSPACE_MODULES))

test-wasm-corpus-light:
	@wasm_exec="$$(go env GOROOT)/lib/wasm/go_js_wasm_exec"; \
	go test -p=1 -skip '^(TestParseCorpusFiles|TestJBalUnitTests|TestJBalUnitBIRTests|TestIntegration|TestBIRSerializationRoundtrip)$$' -timeout 30m ./corpus -exec="$$wasm_exec"

test-wasm-corpus-integration:
	@set -euo pipefail; \
	wasm_exec="$$(go env GOROOT)/lib/wasm/go_js_wasm_exec"; \
	shards="$$(find corpus/bal -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort)"; \
	if [[ -z "$$shards" ]]; then \
		echo "No corpus shards found under corpus/bal" >&2; \
		exit 1; \
	fi; \
	for shard in $$shards; do \
		go test -p=1 -run "^(TestIntegration|TestBIRSerializationRoundtrip)$$/^$${shard}$$" -timeout 30m ./corpus -exec="$$wasm_exec"; \
	done

benchmark-corpus:
	go test -run='^$$' -bench=. -benchtime=1x -timeout 2h ./corpus
