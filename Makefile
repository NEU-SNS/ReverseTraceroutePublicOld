
BIN_DIR=bin

deps:
	go get -d -v github.com/NEU-SNS/ReverseTraceroute/...

protos:
	@ if ! command -v protoc >/dev/null; then \
		echo "protoc must be installed" >&2;\
		exit 1; \
	fi
	go get -u -v github.com/golang/protobuf/protoc-gen-go;
	./scripts/build/buildProto.sh

	@ if [ $$? -ne 0 ]; then \
		echo "failed to build protobuf files" >&2; \
		exit 1; \
	fi

vpservice: 
	go build -a -o $(BIN_DIR)/vpservice ./cmd/vpservice;

revtr:
	go build -a -o $(BIN_DIR)/revtr ./cmd/revtr;

atlas:
	go build -a -o $(BIN_DIR)/atlas ./cmd/atlas;

plcontroller:
	go build -a -o $(BIN_DIR)/plcontroller ./cmd/plcontroller;

controller:
	go build -a -o $(BIN_DIR)/controller ./cmd/controller;

plvp:
	env GOARCH=386 GOOS=linux go build -a -o $(BIN_DIR)/plvp ./cmd/plvp;

all: deps protos vpservice revtr atlas plcontroller controller plvp

.PHONY: \
	deps \
	protos \
	vpservice \
	revtr \
	atlas \
	plcontroller \
	controller \
	plvp \
	all


