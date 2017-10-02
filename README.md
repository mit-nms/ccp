Congestion Control Plane
========================

The congestion control plane allows out-of-the-loop control over congestion control events in various datapaths.
There is one included datapath, in `udpDataplane`, which implements reliable delivery.

There are 4 main parts of this repository:
- An IPC layer (`ipc`) allowing for different IPC backends (`ipcBackend` package interface). Currently unix sockets (`unixsocket`) and netlink sockets (`netlinkipc`) are implented
- A sample UDP datapath with reliable delivery (`udpDataplane`)
    - Note: the UDP datapath does not have full functionality.
- An executable congestion control plane (`ccp`), and interface for defining congestion control schemes (`ccpFlow`)
- Various congestion control schemes (`reno`, `cubic`, `vegas`, etc).

How to run
----------

- `go get ./...`
- `make`
- `./ccpl --datapath=<udp|kernel> --congAlg=<...>`


How to write a new congestion control algorithm 
------------------------------------------------

- Make a new sub-package: `mkdir <my_alg>`
- Define an exported type in your subpackage which implements `ccpFlow.Flow`
- Export a function with signature `func Init();` which calls `ccpFlow.Register()`. It takes:
    - A name for your algorithm (used by the `--congAlg` flag)
    - A closure which returns an instance of your type.
- In `ccp/ccp.go`: 
    - Import your package: `import "ccp/<my_alg>"` 
    - Call `Init()` at the top of `main()`: `<my_alg>.Init()`


Dependencies
------------

- Logrus https://github.com/sirupsen/logrus
