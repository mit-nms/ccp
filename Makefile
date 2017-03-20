PACKAGES = ./ipc \
		   ./ipcBackend \
		   ./udpDataplane \
		   ./unixsocket \
		   ./ccpFlow \
		   ./reno \
		   ./ccp

all: compile test 

compile: ccpl testClient testServer

ccpl: capnpMsg/ccp.capnp.go build
	go build -o ./ccpl ccp/ccp

capnpMsg/ccp.capnp.go:
	mkdir -p ./capnpMsg
	capnp compile -I$(GOPATH)/src/zombiezen.com/go/capnproto2/std -ogo:./capnpMsg ccp.capnp

build:
	go build $(PACKAGES)

test: ccpl testClient testServer
	go test $(PACKAGES)
	go vet $(PACKAGES)
	#golint $(PACKAGES)

testClient: build
	go build -o ./testClient ./tests/testClient/client.go

testServer: build
	go build -o ./testServer ./tests/testServer/server.go

clean:
	rm -rf ./capnpMsg
	rm -f ./testClient
	rm -f ./testServer
	rm -f ./ccpl
