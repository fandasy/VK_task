PROTO_DIR := ./proto
GEN_DIR := .

PROTO_FILE := $(PROTO_DIR)/pubSub.proto

# Protoc command
PROTOC := protoc
PROTOC_FLAGS := -I $(PROTO_DIR)
GO_OUT := --go_out=$(GEN_DIR)
GO_GRPC_OUT := --go-grpc_out=$(GEN_DIR)

.PHONY: all clean generate

all: generate

generate:
	@mkdir -p $(GEN_DIR)
	$(PROTOC) $(PROTOC_FLAGS) $(PROTO_FILE) $(GO_OUT) $(GO_GRPC_OUT)

clean:
	rm -rf $(GEN_DIR)