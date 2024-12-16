GO_LDFLAGS = -ldflags "-s -w"
all: package

.phony: init livhub package clean

init:
	go mod tidy
	go generate ./...

livhub: init
	mkdir -p build/bin
	go build -o build/bin/livhub $(GO_LDFLAGS) cmd/livhub/main.go

package: livhub
	mkdir -p build/config
	cp -n config.yml build/config/config.yml || true

clean:
	rm -rf build

help:
	@echo "make - same as make all"
	@echo "make all - init, build livhub, package"
	@echo "make init - init go mod"
	@echo "make livhub - build livhub"
	@echo "make package - package livhub"
	@echo "make clean - clean build"
	@echo "make help - show help"
