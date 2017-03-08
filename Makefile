PACKAGES = ./ipc \
		   ./ipcBackend \
		   ./mmap \
		   ./udpDataplane \
		   ./unixsocket \
		   ./ccp

all: ccp

ccp: generate build test

generate:
	capnp compile -I$(GOPATH)/src/zombiezen.com/go/capnproto2/std -ogo:./capnpMsg ccp.capnp

build:
	go build $(PACKAGES)

test:
	go test $(PACKAGES)
	go vet $(PACKAGES)
	#golint $(PACKAGES)
