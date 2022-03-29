.PHONY: clean devrun dev

VERSION=0.0.1
RELEASE=1
APP_NAME=all-in-one
FILE_NAME=$(APP_NAME)-$(VERSION)-$(RELEASE).$(shell uname -s).$(shell uname -m)
DEP_OBJS=$(shell find . -name "*.go" -type f)
TARGETS=$(APP_NAME)
CGO_ENABLED=1

all: $(APP_NAME)

$(APP_NAME): $(DEP_OBJS)
	CGO_ENABLED=$(CGO_ENABLED) ./scripts/go-build.sh


devrun: $(APP_NAME)
	@echo "./${APP_NAME} \
		-f ./etc/config.yaml \
		--token-auth-file=./etc/tokens.cvs \
		--anonymous-auth \
		--authorization-mode=AlwaysAllow \
		-v 10 \
		--logtostderr"
		#--tls-cert-file=./run/pki/apiserver.crt \
		#--tls-private-key-file=./run/pki/apiserver.key \

run: $(APP_NAME)
	./$(APP_NAME) \
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



dev: $(DEP_OBJS)
	APP_NAME=$(APP_NAME) watcher --logtostderr -v 3 -e build -e .git -e docs -e vendor -f .go -f .sql -d 1000ms -i ../

clean:
	rm -f $(APP_NAME)

tools:
	go get -u google.golang.org/grpc && \
	go get -u github.com/golang/protobuf/protoc-gen-go
