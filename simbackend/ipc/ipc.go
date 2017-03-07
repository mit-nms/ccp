package ipc

type IpcLayer interface {
	Send(socketId uint32, ackNo uint32) error
	Listen() (chan uint32, error)
	Close() error
}
