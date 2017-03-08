PACKAGES = ./ipc \
		   ./mmap \
		   ./udpDataplane \
		   ./unixsocket

all: ccp

ccp: generate test

generate:
	capnp compile -I$(GOPATH)/src/zombiezen.com/go/capnproto2/std -ogo:./capnpMsg ccp.capnp

test:
	go test $(PACKAGES)
	go vet $(PACKAGES)
	#golint $(PACKAGES)
