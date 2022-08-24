GORELEASER ?= goreleaser
APP_NAME=all-in-one
APP_PATH=./bin/$(APP_NAME)
BUILD_ENVS=$(shell ./scripts/env.sh)
DEP_OBJS=$(shell find . -name "*.go" -type f)
TARGETS=$(APP_PATH)

all: $(TARGETS)

$(APP_PATH): $(DEP_OBJS)
	go build -ldflags="$(shell ./scripts/ldflags.sh)" -o $@ ./cmd/all-in-one

.PHONY: build
build:
	$(BUILD_ENVS) $(GORELEASER) build --single-target --snapshot --rm-dist

# https://goreleaser.com/quick-start/
.PHONY: release
release: goreleaser
	$(BUILD_ENVS) $(GORELEASER) release 

.PHONY: pkg
pkg: goreleaser
	$(BUILD_ENVS) $(GORELEASER) release --snapshot --rm-dist

.PHONY: goreleaser
goreleaser:
	@{ \
		if ! command -v '$(GORELEASER)' >/dev/null 2>/dev/null; then \
			echo >&2 '$(GORELEASER) command not found. Please install goreleaser. https://goreleaser.com/install/'; \
			exit 1; \
		fi \
	}


.PHONY: devrun
devrun: $(APP_NAME)
	@echo "${APP_PATH} \
		-f ./etc/config.yaml \
		--token-auth-file=./etc/tokens.cvs \
		--anonymous-auth \
		--authorization-mode=AlwaysAllow \
		-v 10 \
		--logtostderr"
		#--tls-cert-file=./run/pki/apiserver.crt \
		#--tls-private-key-file=./run/pki/apiserver.key \

.PHONY: run
run: $(APP_PATH)
	$(APP_PATH) \
		-f ./etc/config.yaml \
		-v 10 \
		--cert-dir /tmp/$(APP_NAME) \
		--add-dir-header \
		--logtostderr 2>&1

		#--anonymous-auth \
		#--debug-config --dry-run \
		--authorization-mode=AlwaysAllow \
		#--tls-cert-file=./run/pki/apiserver.crt \
		#--tls-private-key-file=./run/pki/apiserver.key \

.PHONY: docker
docker: 
	docker run --rm -it \
		-v `pwd`/etc/config.yaml:/config.yaml \
		ybbbbasdf/all-in-all:{VERSION} \
		-f /config.yaml
		

.PHONY: dev
dev: $(DEP_OBJS)
	APP_NAME=$(APP_NAME) watcher --logtostderr -v 3 -e bin -e dist -e .git -e docs -e vendor -f .go -f .sql -d 1000ms

.PHONY: clean
clean:
	rm -rf ./bin/* ./dist

tools:
	go get -u google.golang.org/grpc && \
	go get -u github.com/golang/protobuf/protoc-gen-go

.PHONY: status
status:
	./scripts/print-workspace-status.sh
