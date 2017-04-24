using Go = import "/go.capnp";
@0xaecc846565bc5f9c;
$Go.package("capnpMsg");
$Go.import("ccp/capnpMsg");

enum MsgType {
    ack @0;
    cwnd @1;
    create @2;
    drop @3;
}

struct UIntMsg {
    type @0 :MsgType;
    socketId @1 :UInt32;
    val @2 :UInt32;
}

struct UIntStrMsg {
    type @0 :MsgType;
    socketId @1 :UInt32;
    numUInt32 @2 :UInt32;
    val @3 :Text;
}

struct StrMsg {
    type @0 :MsgType;
    socketId @1 :UInt32;
    val @2 :Text;
}

struct UInt32UInt64Msg {
    type @0 :MsgType;
    socketId @1 :UInt32;
    numInt32 @2 :UInt32;
    numInt64 @3 :UInt64;
}
