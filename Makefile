SHELL := /bin/sh
BUF ?= buf
BREAKING_AGAINST ?= .git\#tag=v0.8.0
PROTOC_GEN_GO_VERSION := v1.36.11
PROTOC_GEN_GO_GRPC_VERSION := v1.6.2
PLUGINS := .bin/protoc-gen-go .bin/protoc-gen-go-grpc

.PHONY: tools fmt generate lint policy check breaking

tools: $(PLUGINS)

.bin/protoc-gen-go: Makefile
	GOBIN=$(CURDIR)/.bin go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)

.bin/protoc-gen-go-grpc: Makefile
	GOBIN=$(CURDIR)/.bin go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)

fmt:
	$(BUF) format -w

generate: $(PLUGINS)
	$(BUF) generate

lint:
	$(BUF) format --diff --exit-code
	$(BUF) lint
	$(BUF) build -o /dev/null

# Every RPC must declare (aipay.authz.v1.policy); see internal/policycheck.
policy: generate
	@mkdir -p build
	$(BUF) build -o build/descriptors.binpb
	GOWORK=off go run ./internal/cmd/policycheck build/descriptors.binpb

check: lint generate policy
	GOWORK=off go vet ./...
	GOWORK=off go test ./...

breaking:
	$(BUF) breaking --against '$(BREAKING_AGAINST)'
