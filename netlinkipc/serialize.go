package netlinkipc

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type msgType uint8

const (
	CREATE msgType = iota
	ACK
	DROP
	CWND
)

func readType(b []byte) (m msgType, err error) {
	buf := bytes.NewBuffer(b)
	err = binary.Read(buf, binary.LittleEndian, &m)

	return
}

// (uint32, uint32, string) serialized format
// ----------------------------------------------------------
// | Msg Type | Len (B)  | Uint32    | Uint32    |  String  |
// | (1 B)    | (1 B)    | (32 bits) | (32 bits) |(variable)|
// ----------------------------------------------------------
func writeUInt32AndUInt32AndString(
	typ msgType,
	i uint32,
	j uint32,
	str string,
) ([]byte, error) {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint8(typ))

	if len(str) > 246 {
		return []byte{}, fmt.Errorf("max string length 246: %s", str)
	}

	binary.Write(buf, binary.LittleEndian, uint8(len(str)+10))

	binary.Write(buf, binary.LittleEndian, i)
	binary.Write(buf, binary.LittleEndian, j)
	binary.Write(buf, binary.LittleEndian, []byte(str))

	return buf.Bytes(), nil
}

func readUInt32AndUInt32AndString(b []byte) (
	typ msgType,
	i uint32,
	j uint32,
	strn string,
) {
	buf := bytes.NewBuffer(b)
	var msgLen uint8
	var str []byte

	binary.Read(buf, binary.LittleEndian, &typ)
	binary.Read(buf, binary.LittleEndian, &msgLen)
	str = make([]byte, msgLen-10)
	binary.Read(buf, binary.LittleEndian, &i)
	binary.Read(buf, binary.LittleEndian, &j)
	binary.Read(buf, binary.LittleEndian, &str)

	// remove null terminator
	if str[len(str)-1] == byte(0) {
		str = str[:len(str)-1]
	}
	strn = string(str)

	return
}

// (uint32, string) serialized format
// -----------------------------------------------
// | Msg Type | Len (B)  | Uint32    | String    |
// | (1 B)    | (1 B)    | (32 bits) | (variable)|
// -----------------------------------------------
func writeUInt32AndString(
	typ msgType,
	i uint32,
	str string,
) ([]byte, error) {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint8(typ))

	if len(str) > 249 {
		return []byte{}, fmt.Errorf("max string length 249: %s", str)
	}

	binary.Write(buf, binary.LittleEndian, uint8(len(str)+6))

	binary.Write(buf, binary.LittleEndian, i)
	binary.Write(buf, binary.LittleEndian, []byte(str))

	return buf.Bytes(), nil
}

func readUInt32AndString(b []byte) (
	typ msgType,
	i uint32,
	strn string,
) {
	buf := bytes.NewBuffer(b)
	var msgLen uint8
	var str []byte

	binary.Read(buf, binary.LittleEndian, &typ)
	binary.Read(buf, binary.LittleEndian, &msgLen)
	str = make([]byte, msgLen-6)
	binary.Read(buf, binary.LittleEndian, &i)
	binary.Read(buf, binary.LittleEndian, &str)

	// remove null terminator
	if str[len(str)-1] == byte(0) {
		str = str[:len(str)-1]
	}
	strn = string(str)

	return
}

// (uint32, uint32) serialized format
// -----------------------------------------------
// | Msg Type | Len (B)  | Uint32    | Uint32    |
// | (1 B)    | (1 B)    | (32 bits) | (32 bits) |
// -----------------------------------------------
func writeUInt32AndUInt32(
	typ msgType,
	i uint32,
	j uint32,
) ([]byte, error) {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint8(typ))
	binary.Write(buf, binary.LittleEndian, uint8(10))

	binary.Write(buf, binary.LittleEndian, i)
	binary.Write(buf, binary.LittleEndian, j)

	return buf.Bytes(), nil
}

func readUInt32AndUInt32(b []byte) (
	typ msgType,
	i uint32,
	j uint32,
) {
	buf := bytes.NewBuffer(b)
	var len uint8

	binary.Read(buf, binary.LittleEndian, &typ)
	binary.Read(buf, binary.LittleEndian, &len)
	binary.Read(buf, binary.LittleEndian, &i)
	binary.Read(buf, binary.LittleEndian, &j)

	return
}

// (uint32, uint32, int64) serialized format
// ----------------------------------------------------------
// | Msg Type | Len (B)  | Uint32    | Uint32    | Uint64    |
// | (1 B)    | (1 B)    | (32 bits) | (32 bits) | (64 bits) |
// ----------------------------------------------------------
func writeUInt32AndUInt32AndUInt64(typ msgType, i uint32, j uint32, k uint64) ([]byte, error) {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint8(typ))
	binary.Write(buf, binary.LittleEndian, uint8(18))

	binary.Write(buf, binary.LittleEndian, i)
	binary.Write(buf, binary.LittleEndian, j)
	binary.Write(buf, binary.LittleEndian, k)

	return buf.Bytes(), nil
}

func readUInt32AndUInt32AndUInt64(b []byte) (
	typ msgType,
	i uint32,
	j uint32,
	k uint64,
) {
	buf := bytes.NewBuffer(b)
	var len uint8

	binary.Read(buf, binary.LittleEndian, &typ)
	binary.Read(buf, binary.LittleEndian, &len)
	binary.Read(buf, binary.LittleEndian, &i)
	binary.Read(buf, binary.LittleEndian, &j)
	binary.Read(buf, binary.LittleEndian, &k)

	return
}
