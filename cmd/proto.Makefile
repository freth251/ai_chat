# t
gen-proto: generate-pb-files chsi-protoc csi-protoc usi-protoc


INTERFACES_DIR := $(REPO_DIR)/interfaces
CHSI_DIR := $(WORKDIR)/chat-history-service
CSI_DIR := $(WORKDIR)/chat-service
USI_DIR := $(WORKDIR)/user-service

generate-pb-files: 
    $(shell mkdir -p $(dir $(CHSI_DIR)/chsi_pb))
    $(shell mkdir -p $(dir $(CSI_DIR)/csi_pb))
    $(shell mkdir -p $(dir $(USI_DIR)/usi_pb))

chsi-protoc:
    protoc $(INTERFACES_DIR)/chat-history-service-interface/chsi.proto \
    --proto_path=$(INTERFACES_DIR)/chat-history-service-interface/ \
    --go_out=$(CHSI_DIR) \
    --go-grpc=$(CHSI_DIR)

csi-protoc:
    protoc $(INTERFACES_DIR)/chat-service-interface/csi.proto \
    --proto_path=$(INTERFACES_DIR)/chat-service-interface/ \
    --go_out=$(CSI_DIR) \
    --go-grpc=$(CSI_DIR)

usi-protoc:
    protoc $(INTERFACES_DIR)/user-service-interface/usi.proto \
    --proto_path=$(INTERFACES_DIR)/user-service-interface/ \
    --go_out=$(USI_DIR) \
    --go-grpc=$(USI_DIR)