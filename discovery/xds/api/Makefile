PROTOC_VERSION := 3.14.0
GOLANG_PROTOBUF_VERSION := v1.4.3
PROTOC_PGV_VERSION := v0.4.1

GOPATH_DIR := $(shell go env GOPATH | awk -F: '{print $$1}')
PROTO_DIR := .

define go_mod_latest_version
	find $(GOPATH_DIR)/pkg/mod/$(1) -maxdepth 1 -name '$(2)@*' | xargs -L 1 basename | sort -r | head -1 | awk -F@ '{print $$2}'
endef

DATAPLANE_API_LATEST_VERSION := master
DATAPLANE_API_ACTUAL_VERSION := $(shell $(call go_mod_latest_version,github.com/envoyproxy,data-plane-api))
UDPA_LATEST_VERSION := master
UDPA_ACTUAL_VERSION := $(shell $(call go_mod_latest_version,github.com/cncf,udpa))
GOOGLEAPIS_LATEST_VERSION := master
GOOGLEAPIS_ACTUAL_VERSION := $(shell $(call go_mod_latest_version,github.com/googleapis,googleapis))

protoc_search_go_packages := \
	github.com/golang/protobuf@$(GOLANG_PROTOBUF_VERSION) \
	github.com/envoyproxy/protoc-gen-validate@$(PROTOC_PGV_VERSION) \
	github.com/envoyproxy/data-plane-api@$(DATAPLANE_API_ACTUAL_VERSION) \
	github.com/cncf/udpa@$(UDPA_ACTUAL_VERSION) \
	github.com/googleapis/googleapis@$(GOOGLEAPIS_ACTUAL_VERSION)

protoc_search_go_paths := $(foreach go_package,$(protoc_search_go_packages),--proto_path=$(GOPATH_DIR)/pkg/mod/$(go_package))

go_import_mapping_entries := \
	google/protobuf/any.proto=github.com/golang/protobuf/ptypes/any \
	google/protobuf/duration.proto=github.com/golang/protobuf/ptypes/duration \
	google/protobuf/struct.proto=github.com/golang/protobuf/ptypes/struct \
	google/protobuf/timestamp.proto=github.com/golang/protobuf/ptypes/timestamp \
	google/protobuf/wrappers.proto=github.com/golang/protobuf/ptypes/wrappers \
	envoy/service/discovery/v3/discovery.proto=github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3

# see https://makefiletutorial.com/
comma := ,
empty:=
space := $(empty) $(empty)

go_mapping_with_spaces := $(foreach entry,$(go_import_mapping_entries),M$(entry),)
go_mapping := $(subst $(space),$(empty),$(go_mapping_with_spaces))

PROTOC_GO := protoc \
	--proto_path=./generated/include \
	--proto_path=. \
	$(protoc_search_go_paths) \
	--go_opt=paths=source_relative \
	--go_out=plugins=grpc,$(go_mapping):.

.PHONY: api/clean
api/clean: api/clean/proto

.PHONY: api/clean/proto
api/clean/proto: ## Dev: Remove auto-generated Protobuf files
	find $(PROTO_DIR) -wholename 'api/*.pb.go' -delete
	find $(PROTO_DIR) -wholename 'api/*.pb.validate.go' -delete

.PHONY: api/generate
api/generate: api/clean/proto api/protoc/v1alpha1

.PHONY: api/install
api/install: api/install/protoc-gen-go api/install/protoc-gen-validate api/install/proto-deps

.PHONY: api/install/protoc-gen-go
api/install/protoc-gen-go: ## Bootstrap: Install Protoc Go Plugin (protobuf Go generator)
	go get github.com/golang/protobuf/protoc-gen-go@$(GOLANG_PROTOBUF_VERSION)

.PHONY: api/install/protoc-gen-validate
api/install/protoc-gen-validate: ## Bootstrap: Install Protoc Gen Validate Plugin (protobuf validation code generator)
	go get github.com/envoyproxy/protoc-gen-validate@$(PROTOC_PGV_VERSION)

.PHONY: api/install/proto-deps
api/install/proto-deps:
	go get github.com/golang/protobuf@$(GOLANG_PROTOBUF_VERSION) \
           	github.com/envoyproxy/protoc-gen-validate@$(PROTOC_PGV_VERSION) \
           	github.com/envoyproxy/data-plane-api@$(DATAPLANE_API_LATEST_VERSION) \
           	github.com/cncf/udpa@$(UDPA_LATEST_VERSION) \
           	github.com/googleapis/googleapis@$(GOOGLEAPIS_LATEST_VERSION)

.PHONY: api/protoc/v1alpha1
api/protoc/v1alpha1:
	$(PROTOC_GO) api/v1alpha1/*.proto

