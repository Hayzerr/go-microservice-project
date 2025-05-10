PROTO_DIR=proto
PROTO_OUT=pb

all: build

build:
	go mod tidy
	go work sync
	go vet ./...

proto:
	protoc \
		--go_out=$(PROTO_OUT) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto
