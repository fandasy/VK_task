PROTO_DIR := ./proto
GEN_DIR := ./gen

PROTO_FILE := $(PROTO_DIR)/pubSub.proto

# Protoc command
PROTOC := protoc
PROTOC_FLAGS := -I $(PROTO_DIR)
GO_OUT := --go_out=$(GEN_DIR) --go_opt=paths=source_relative
GO_GRPC_OUT := --go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative

.PHONY: all clean generate

all: generate

generate:
	@mkdir -p $(GEN_DIR)
	$(PROTOC) $(PROTOC_FLAGS) $(PROTO_FILE) $(GO_OUT) $(GO_GRPC_OUT)

clean:
	rm -rf $(GEN_DIR)