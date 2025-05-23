package amqpproxy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/richardpark-msft/amqpfaultinjector/internal/logging"
	"github.com/richardpark-msft/amqpfaultinjector/internal/shared"
	"github.com/richardpark-msft/amqpfaultinjector/internal/utils"
)

type AMQPProxy struct {
	nextFileID                    uint64
	conn                          atomic.Pointer[net.Listener]
	localEndpoint, remoteEndpoint string
	options                       AMQPProxyOptions
}

type AMQPProxyOptions struct {
	LogFolder      string
	EnableJSON     bool
	EnableHexDumps bool

	TLSKeyLogFile string

	// Folder where a certificate, for our TLS endpoint, is stored. If no certificate is present it is
	// generated.
	CertDir string

	// DisableTLSForLocalEndpoint will disable TLS for the _local_ endpoint, while still using TLS
	// when communicating with the remote host. This can be used an alternative to accepting self-signed
	// certificates.
	// NOTE: even with this flag set to true, no traffic between your machine and Azure is unencrypted.
	DisableTLSForLocalEndpoint bool

	DisableStateTracing bool

	TransformerOptions *logging.TransformerOptions
}

// localEndpoint is the endpoint that the proxy will listen on.
// remoteEndpoint is the endpoint that the proxy will connect to.
func NewAMQPProxy(localEndpoint, remoteEndpoint string, options *AMQPProxyOptions) (*AMQPProxy, error) {
	// okay, all we're going to do is just intersperse ourselves between the remote service and another client that's connecting.
	if localEndpoint == "" {
		panic("localEndpoint is not set")
	}

	if remoteEndpoint == "" {
		panic("remoteEndpoint is not set")
	}

	// can override for emulator
	if !strings.Contains(remoteEndpoint, ":") {
		remoteEndpoint += ":5671"
	}

	if options == nil {
		options = &AMQPProxyOptions{}
	}

	amqpProxy := &AMQPProxy{
		localEndpoint:  localEndpoint,
		remoteEndpoint: remoteEndpoint,
		options:        *options,
	}

	return amqpProxy, nil
}

func (fi *AMQPProxy) Close() error {
	listener := fi.conn.Swap(nil)

	if listener != nil {
		return (*listener).Close()
	}

	return nil
}

func (proxy *AMQPProxy) ListenAndServe() error {
	slog.Info("Starting server...")

	listener, err := net.Listen("tcp4", proxy.localEndpoint)

	if err != nil {
		return err
	}

	certFile, keyFile, cert, err := shared.LoadOrCreateCert(proxy.options.CertDir)

	if err != nil {
		return err
	}

	slog.Info("Certificate information:", "cert", certFile, "key", keyFile)

	var tlsKeyLogWriter io.Writer

	if proxy.options.TLSKeyLogFile != "" {
		tmpWriter, err := os.OpenFile(proxy.options.TLSKeyLogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)

		if err != nil {
			return err
		}

		defer utils.CloseWithLogging("tlskeylogfile", tmpWriter)
		tlsKeyLogWriter = tmpWriter
	}

	if !proxy.options.DisableTLSForLocalEndpoint {
		listener = tls.NewListener(listener, &tls.Config{
			Certificates: []tls.Certificate{
				cert,
			},
		})
	}

	defer utils.CloseWithLogging("tls Listener", listener)

	slog.Info("Server started, listening for connections...")

	for {
		// a client has connected to our listening socket
		localConn, err := listener.Accept()

		if err != nil {
			slog.Error("Connection failed to accept", "err", err)
			return err
		}

		fn := func() error {
			defer utils.CloseWithLogging("local "+localConn.RemoteAddr().String(), localConn)
			slog.Info("Connection started", "clientip", localConn.RemoteAddr())

			// open up connection to remote host
			remoteConn, err := net.Dial("tcp4", proxy.remoteEndpoint)

			if err != nil {
				slog.Error("Failed to open remote connection", "endpoint", proxy.remoteEndpoint, "err", err)
				return err
			}

			defer utils.CloseWithLogging("remote", remoteConn)
			slog.Info("Setting up remote TLS connection", "remote", proxy.remoteEndpoint)

			remoteConn = tls.Client(remoteConn, &tls.Config{
				ServerName: utils.HostOnly(proxy.remoteEndpoint),
				// TODO: not thread safe....
				KeyLogWriter: tlsKeyLogWriter,
			})

			connectionIndex := atomic.AddUint64(&proxy.nextFileID, 1)

			ml := &logging.MultiLogger{}

			if proxy.options.EnableJSON {
				path := filepath.Join(proxy.options.LogFolder, fmt.Sprintf("amqpproxy-traffic-%d.json", connectionIndex))
				tmp, err := logging.NewJSONLogger(path, !proxy.options.DisableStateTracing, proxy.options.TransformerOptions)

				if err != nil {
					return err
				}

				ml.Loggers = append(ml.Loggers, tmp)
			}

			if proxy.options.EnableHexDumps {
				path := filepath.Join(proxy.options.LogFolder, fmt.Sprintf("amqpproxy-hexdump-%d.txt", connectionIndex))
				tmp, err := logging.NewHexDumpLogger(path)

				if err != nil {
					return err
				}

				ml.Loggers = append(ml.Loggers, tmp)
			}

			if len(ml.Loggers) > 0 {
				defer utils.CloseWithLogging("multilogger", ml)
			}

			ctx, cancel := context.WithCancelCause(context.Background())

			go func() {
				if err := proxy.mirrorConn(true, localConn, remoteConn, ml); err != nil {
					cancel(err)
				}
			}()

			go func() {
				if err := proxy.mirrorConn(false, remoteConn, localConn, ml); err != nil {
					cancel(err)
				}
			}()

			<-ctx.Done()

			slog.Info("Connection mirroring failed", "clientip", localConn.RemoteAddr(), "err", context.Cause(ctx))
			utils.CloseWithLogging(localConn.RemoteAddr().String(), localConn)

			return nil
		}

		go func() {
			err := fn()

			if err != nil {
				slog.Error("failure in service", "error", err)
			}
		}()
	}
}

type conn interface {
	io.Writer
	RemoteAddr() net.Addr
}

func (proxy *AMQPProxy) mirrorConn(out bool, source io.Reader, dest conn, logger logging.Logger) error {
	label := "in"

	if out {
		label = "out"
	}

	// TODO: hierarchy?
	slogger := slog.Default().With("label", label, "remoteaddr", dest.RemoteAddr().String())

	connBytes := make([]byte, 1024*1024)
	disconnect := false

loop:
	for !disconnect {
		n, err := source.Read(connBytes)

		switch {
		case errors.Is(err, io.EOF):
			// if there are still some bytes, process those, and then we're done.
			if n == 0 {
				break loop
			}

			// some data came in, let that get processed
			disconnect = true
		case err != nil:
			slogger.Error("Failed to read from connection", "err", err)
			return err
		}

		packet := connBytes[0:n]

		if err := logger.AddPacket(out, packet); err != nil {
			slogger.Error("Failed to write packet to log", "error", err)
		}

		if _, err := dest.Write(packet); err != nil {
			slogger.Error("Failed to write to remote endpoint", "error", err)
			return err
		}
	}

	slogger.Info("Exiting loop")
	return nil
}
