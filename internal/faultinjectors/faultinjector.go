package faultinjectors

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync/atomic"

	"github.com/richardpark-msft/amqpfaultinjector/internal/logging"
	"github.com/richardpark-msft/amqpfaultinjector/internal/proto/frames"
	"github.com/richardpark-msft/amqpfaultinjector/internal/shared"
	"github.com/richardpark-msft/amqpfaultinjector/internal/utils"
)

// TODO: some more factoring to make the AMQPProxy and FaultInjector share a bit more code would be good.
type FaultInjector struct {
	conn                          atomic.Pointer[net.Listener]
	localEndpoint, remoteEndpoint string
	options                       FaultInjectorOptions
	callback                      MirrorCallback
	tlsKeyLogWriter               io.Writer
	frameLogger                   *logging.FrameLogger
	closedByUser                  atomic.Bool

	serverCtx    context.Context
	cancelServer context.CancelFunc
}

type FaultInjectorOptions struct {
	TLSKeyLogFile string
	JSONLFile     string

	AddressFile string

	// Folder where a certificate, for our TLS endpoint, is stored. If no certificate is present it is
	// generated.
	CertDir string
}

func NewFaultInjector(localEndpoint, remoteEndpoint string, injector MirrorCallback, options *FaultInjectorOptions) (*FaultInjector, error) {
	// okay, all we're going to do is just intersperse ourselves between the remote service and another client that's connecting.
	if localEndpoint == "" {
		panic("localEndpoint is not set")
	}

	if remoteEndpoint == "" {
		panic("remoteEndpoint is not set")
	}

	if !strings.Contains(remoteEndpoint, ":") {
		remoteEndpoint += ":5671"
	}

	if options == nil {
		options = &FaultInjectorOptions{}
	}

	serverCtx, cancelServer := context.WithCancel(context.Background())

	fi := &FaultInjector{
		localEndpoint:  localEndpoint,
		remoteEndpoint: remoteEndpoint,
		callback:       injector,
		options:        *options,

		serverCtx:    serverCtx,
		cancelServer: cancelServer,
	}

	if options.JSONLFile != "" {
		fl, err := logging.NewFrameLogger(options.JSONLFile)

		if err != nil {
			utils.Panicf("failed creating framelogger at %s: %w", options.JSONLFile, err)
		}

		fi.frameLogger = fl
	}

	return fi, nil
}

func (fi *FaultInjector) Close() error {
	if fi.closedByUser.CompareAndSwap(false, true) {
		fi.cancelServer()

		listener := fi.conn.Swap(nil)

		if listener != nil {
			return (*listener).Close()
		}
	}

	return nil
}

func (fi *FaultInjector) ListenAndServe() error {
	slog.Info("Starting server...")
	listener, err := net.Listen("tcp4", fi.localEndpoint)

	if err != nil {
		return err
	}

	fi.conn.Store(&listener)

	if fi.options.AddressFile != "" {
		if err := os.WriteFile(fi.options.AddressFile, []byte(listener.Addr().String()), 0777); err != nil {
			return fmt.Errorf("failed to create file to write address file at %s: %w", fi.options.AddressFile, err)
		}

		defer os.Remove(fi.options.AddressFile)
	}

	slog.Info("Listener started", "address", listener.Addr().String())

	certFile, keyFile, cert, err := shared.LoadOrCreateCert(fi.options.CertDir)

	if err != nil {
		return err
	}

	slog.Info("Certificate information:", "cert", certFile, "key", keyFile)

	if fi.options.TLSKeyLogFile != "" {
		tmpWriter, err := os.OpenFile(fi.options.TLSKeyLogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)

		if err != nil {
			return err
		}

		defer utils.CloseWithLogging("tlskeylogfile", tmpWriter)

		// TODO: this will be used by multiple goroutines, by their individual tls.Client's. We might
		// need to do some work to make it goroutine-safe.
		fi.tlsKeyLogWriter = tmpWriter
	}

	listener = tls.NewListener(listener, &tls.Config{
		Certificates: []tls.Certificate{
			cert,
		},
	})

	defer func() {
		if !fi.closedByUser.Load() {
			utils.CloseWithLogging("tls.Listener", listener)
		}
	}()

	slog.Info("Server started, listening for connections...")

	for {
		localConn, err := listener.Accept()

		if err != nil {
			if !fi.closedByUser.Load() {
				slog.Error("Connection failed to accept", "err", err)
				return err
			}

			return nil
		}

		go func() {
			if err := fi.mirrorConn(localConn); err != nil {
				slog.Error("Failure when mirroring connection", "endpoint", fi.remoteEndpoint, "err", err)
			}
		}()
	}
}

// ListenAddr is the address that the fault injector is listening on, including the port (ex: 127.0.0.1:39607)
// If the service's endpoint has not yet started this function returns an empty string.
func (fi *FaultInjector) ListenAddr() string {
	listener := fi.conn.Load()

	if listener == nil {
		return ""
	}

	return (*listener).Addr().String()
}

func (fi *FaultInjector) mirrorConn(localNetConn net.Conn) error {
	defer utils.CloseWithLogging("local"+localNetConn.RemoteAddr().String(), localNetConn)
	slog.Info("Connection started", "clientip", localNetConn.RemoteAddr())

	// open up connection to remote host
	remoteNetConn, err := net.Dial("tcp4", fi.remoteEndpoint)

	if err != nil {
		return fmt.Errorf("failed to mirror connection: %w", err)
	}

	defer utils.CloseWithLogging("remote", remoteNetConn)
	slog.Info("Setting up remote TLS connection", "remote", fi.remoteEndpoint)

	remoteTLSConn := tls.Client(remoteNetConn, &tls.Config{
		ServerName:   utils.HostOnly(fi.remoteEndpoint),
		KeyLogWriter: fi.tlsKeyLogWriter,
	})

	localConn := frames.NewConnReadWriter(localNetConn)
	remoteConn := frames.NewConnReadWriter(remoteTLSConn)

	// run the mirroring logic until the connection is passed the OPEN frames.
	if err := Mirror(fi.serverCtx, MirrorParams{
		Callback:    mirrorConnUntilOpenFrame,
		FrameLogger: fi.frameLogger,
		Local:       localConn,
		Remote:      remoteConn,
	}); err != nil {
		return fmt.Errorf("failed mirroring till the OPEN frame: %w", err)
	}

	// from this point we run the user's callback
	if err := Mirror(fi.serverCtx, MirrorParams{
		Callback:    fi.callback,
		FrameLogger: fi.frameLogger,
		Local:       localConn,
		Remote:      remoteConn,
	}); err != nil {
		return fmt.Errorf("failed mirroring using the user's callback: %w", err)
	}

	return nil
}

func mirrorConnUntilOpenFrame(ctx context.Context, params MirrorCallbackParams) ([]MetaFrame, error) {
	retFrames := []MetaFrame{{Action: MetaFrameActionPassthrough, Frame: params.Frame}}

	if _, isOpenFrame := params.Frame.Body.(*frames.PerformOpen); isOpenFrame {
		return retFrames, io.EOF
	}

	return retFrames, nil
}
