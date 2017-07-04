Congestion Control Plane
========================

The congestion control plane allows out-of-the-loop control over congestion control events in various datapaths.
There is one included datapath, in `udpDataplane`, which implements reliable delivery.

There are 4 main parts of this repository
- An IPC layer (`ipc`) allowing for different IPC backends (`ipcBackend` package interface). Currently unix sockets (`unixsocket`) and netlink sockets (`netlinkipc`) are implented
- A sample UDP datapath with reliable delivery (`udpDataplane`)
- An executable congestion control plane (`ccp`), and interface for defining congestion control schemes (`ccpFlow`)
- Various congestion control schemes (`reno`, `cubic`, `vegas`, etc).

Dependencies
------------

- Logrus https://github.com/sirupsen/logrus
