using Go = import "/go.capnp";
@0xaecc846565bc5f9c;
$Go.package("capnpMsg");
$Go.import("ccp/capnpMsg");

enum UIntMsgType {
    ack @0;
    cwnd @1;
}

struct UIntMsg {
    type @0 :UIntMsgType;
    socketId @1 :UInt32;
    val @2 :UInt32;
}
