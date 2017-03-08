using Go = import "/go.capnp";
@0xaecc846565bc5f9c;
$Go.package("capnpMsg");
$Go.import("ccp/capnpMsg");

enum MsgType {
    ack @0;
    cwnd @1;
    create @2;
}

struct UIntMsg {
    type @0 :MsgType;
    socketId @1 :UInt32;
    val @2 :UInt32;
}

struct StrMsg {
    type @0 :MsgType;
    socketId @1 :UInt32;
    val @2 :Text;
}
