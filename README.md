Congestion Control Plane
========================

The congestion control plane allows out-of-the-loop control over congestion control events in various datapaths.
There is one included datapath, in `udpDataplane`, which implements reliable delivery.

There are 4 main parts of this repository
- An IPC layer (`ipc`) allowing for different IPC backends (`ipcBackend` package interface). Currently unix sockets (`unixsocket`) are implented
- A sample UDP datapath with reliable delivery (`udpDataplane`)
- An executable congestion control plane (`ccp`), and interface for defining congestion control schemes (`ccpFlow`)
- A sample congestion control scheme (`reno`).

Dependencies
------------

- Captain Proto: https://capnproto.org/install.html
- Go bindings for Captain Proto: https://github.com/zombiezen/go-capnproto2
- Logrus https://github.com/sirupsen/logrus
