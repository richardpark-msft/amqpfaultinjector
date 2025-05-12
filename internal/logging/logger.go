package logging

type Logger interface {
	AddPacket(out bool, packet []byte) error
	Close() error
}
