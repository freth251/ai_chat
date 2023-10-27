
BINARY_NAME := ai-chat-services
OUT_BIN_DIR := bin


export REPO_DIR := $(shell git rev-parse --show-toplevel)
export WORKDIR := $(shell pwd)

.PHONY: generate

generate: generate_proto

generate_proto: 
	@$(MAKE) -f proto.Makefile gen-proto


.PHONY: clean

clean:
	go clean -modcache
	rm -rf $(OUT_BIN_DIR)
	find $(WORKDIR) -type f -name '*pb.go' | xargs rm -rf

.PHONY: build
build: $(OUT_BIN_DIR)/$(BINARY_NAME)

$(OUT_BIN_DIR)/$(BINARY_NAME): generate
	go build -o $(OUT_BIN_DIR)/$(BINARY_NAME)