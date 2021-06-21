#!/usr/bin/env bash
#
# Generate all protobuf bindings.
# Run from repository root.
set -e
set -u

if ! [[ "$0" =~ scripts/genproto.sh ]]; then
	echo "must be run from repository root"
	exit 255
fi

if ! [[ $(buf --version) =~ 0.43.2 ]]; then
	echo "could not find buf 0.43.2, is it installed + in PATH?"
	exit 255
fi

# Since we run go get, go mod download, the go.sum will change.
# Make a backup.
cp go.sum go.sum.bak

INSTALL_PKGS=(
  "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway"
  "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger"
)
for pkg in "${INSTALL_PKGS[@]}"; do
    echo "installing $pkg"
    GO111MODULE=on go install "$pkg"
done

GET_PKGS=(
  "google.golang.org/protobuf/cmd/protoc-gen-go@v1.26.0"
  "google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1.0"
  "github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@v0.0.0-20210616093554-9236f7c7b8ca"
)
for pkg in "${GET_PKGS[@]}"; do
    echo "getting $pkg"
    GO111MODULE=on go get "$pkg"
done

buf generate --verbose

mv go.sum.bak go.sum
