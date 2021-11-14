.PHONY: clean devrun dev

VERSION=0.0.1
RELEASE=1
APP_NAME=all
FILE_NAME=$(APP_NAME)-$(VERSION)-$(RELEASE).$(shell uname -s).$(shell uname -m)
DEP_OBJS=$(shell find . -name "*.go" -type f) \
	 ./tracing/tracing.pb.go
TARGETS=$(APP_NAME)
CGO_ENABLED?=1

GO_BUILD_LDFLAGS_CMD      := $(abspath ./scripts/go-build-ldflags.sh)
GO_BUILD_LDFLAGS          := $(shell $(GO_BUILD_LDFLAGS_CMD) LDFLAG)

all: $(APP_NAME)

$(APP_NAME): $(DEP_OBJS)
	GO111MODULE=on CGO_ENABLED=$(CGO_ENABLED) GOARCH=amd64 \
		    go build -ldflags '$(GO_BUILD_LDFLAGS)' -o $@  *.go


.PHONY: devrun run dev
devrun:
	@echo "./${APP_NAME} \
		-f ./etc/${APP_NAME}.yaml \
		--token-auth-file=./etc/tokens.cvs \
		-v 10 \
		--logtostderr"
		#--authorization-mode=AlwaysAllow \
		#--tls-cert-file=./run/pki/apiserver.crt \
		#--tls-private-key-file=./run/pki/apiserver.key \

run:
	./all \
		-f ./etc/all.yaml \
		-v 9 \
		--cert-dir /tmp/all \
		--add-dir-header \
		--logtostderr 2>&1
		# --debug-config --dry-run \
		#--token-auth-file=./run/tokens.cvs \
		## --authorization-mode=AlwaysAllow \
		#--authorization-mode=RBAC \
		#--tls-cert-file=./run/pki/apiserver.crt \
		#--tls-private-key-file=./run/pki/apiserver.key \



dev: $(DEP_OBJS)
	APP_NAME=$(APP_NAME) watcher --logtostderr -v 3 -e build -e .git -e docs -e vendor -f .go -f .sql -d 1000ms -i ../

clean:
	rm -f $(APP_NAME)

tools:
	go get -u google.golang.org/grpc && \
	go get -u github.com/golang/protobuf/protoc-gen-go
	# https://grpc.io/docs/quickstart/go/


%.pb.go: %.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$<
