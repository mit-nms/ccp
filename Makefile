PACKAGES = ./ipc \
		   ./ipcBackend \
		   ./udpDataplane \
		   ./unixsocket \
		   ./netlinkipc \
		   ./ccpFlow \
		   ./reno \
		   ./vegas \
		   ./cubic \
		   ./nl_userapp \
		   ./ccp

all: compile test 

compile: ccpl testClient testServer nltest

ccpl: build
	go build -o ./ccpl ccp/ccp

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

nltest: build
	go build -o ./nltest ccp/nl_userapp

clean:
	rm -f ./testClient
	rm -f ./testServer
	rm -f ./nltest
	rm -f ./ccpl
