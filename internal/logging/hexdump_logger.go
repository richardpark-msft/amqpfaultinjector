package logging

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// This logger writes out a hexdump that Wireshark can import.
//
// This link describes the format:
// https://www.wireshark.org/docs/wsug_html_chunked/ChIOImportSection.html#:~:text=5.5.1.%C2%A0Standard%20ASCII%20Hexdumps
//
// Instructions on how to import this file are in the [importing-amqpproxy-hexdumps-into-wireshark.md] folder
//
// importing-amqpproxy-hexdumps-into-wireshark.md: https://github.com/richardpark-msft/amqpfaultinjector/docs/importing-amqpproxy-hexdumps-into-wireshark.md
type HexDumpLogger struct {
	f *os.File
}

func NewHexDumpLogger(path string) (*HexDumpLogger, error) {
	writer, err := os.Create(path)

	if err != nil {
		return nil, err
	}

	return &HexDumpLogger{f: writer}, nil
}

// AddPacket adds this packet to the hexdump. It uses [time.Now().UTC()] as the timestamp
// for the packet.
func (hdl *HexDumpLogger) AddPacket(out bool, packet []byte) error {
	text, err := dumpHexBytes(time.Now().UTC(), out, packet)

	if err != nil {
		return err
	}

	if _, err := hdl.f.WriteString(text + "\n"); err != nil {
		return err
	}

	return nil
}

func (hdl *HexDumpLogger) Close() error {
	return hdl.f.Close()
}

func dumpHexBytes(timestamp time.Time, out bool, packet []byte) (string, error) {
	builder := strings.Builder{}

	var direction string

	if out {
		direction = "O"
	} else {
		direction = "I"
	}

	builder.WriteString(fmt.Sprintf("%s %s\n", direction, timestamp.Format(time.RFC3339)))

	// now, we'll iterate through the bytes, and dump them into the format that wireshark automatically understands.
	for i := 0; i < len(packet); i += 16 {
		var buff []byte

		if (i + 16) > len(packet) {
			buff = packet[i:]
		} else {
			buff = packet[i : i+16]
		}

		// offset
		builder.WriteString(fmt.Sprintf("%06X ", i))

		// hex string
		for _, b := range buff {
			builder.WriteString(fmt.Sprintf("%02X ", b))
		}

		// bytes, as ASCII
		for _, b := range buff {
			if b < 32 || b > 126 {
				b = '.'
			}
			builder.WriteString(fmt.Sprintf("%c", b))
		}

		builder.WriteString("\n")
	}

	return builder.String(), nil
}
