using Go = import "/go.capnp";
@0xaecc846565bc5f9c;
$Go.package("capnpMsg");
$Go.import("ccp/capnpMsg");

struct NotifyAckMsg {
    socketId @0 :UInt32;
    ackNo @1 :UInt32;
}

struct SetCwndMsg {
    socketId @0 :UInt32;
    cwnd @1 :UInt32;
}
