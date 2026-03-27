BINARY=terraform-provider-purefb
OS_ARCH=linux_amd64
VERSION=0.1.0
PLUGIN_DIR=~/.terraform.d/plugins/registry.terraform.io/purestorage/purefb/$(VERSION)/$(OS_ARCH)

.PHONY: build install test testacc clean fmt

build:
	go build -o $(BINARY)

install: build
	mkdir -p $(PLUGIN_DIR)
	cp $(BINARY) $(PLUGIN_DIR)/

test:
	go test ./internal/... -v

testacc:
	TF_ACC=1 go test ./internal/... -v -timeout 120m

clean:
	rm -f $(BINARY)

fmt:
	gofmt -s -w .
