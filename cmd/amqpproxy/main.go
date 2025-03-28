package main

import (
	"context"
	"path/filepath"

	"github.com/richardpark-msft/amqpfaultinjector/cmd/internal"
	"github.com/richardpark-msft/amqpfaultinjector/internal/amqpproxy"
	"github.com/richardpark-msft/amqpfaultinjector/internal/logging"
	"github.com/spf13/cobra"
)

func main() {
	cmd := newAMQPProxyCommand(context.Background())

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func newAMQPProxyCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use: "amqpproxy",
	}

	internal.AddCommonFlags(cmd)

	disableStateTracking := cmd.Flags().Bool("disable-state-tracing", false, "Disables state tracing - useful if you are experiencing problems or intentionally creating invalid AMQP traffic but still want logging.")
	excludePayloadData := cmd.Flags().Bool("exclude-payload-data", false, "Excludes the data in the payload from logging - useful if you're sending/receiving large messages and trying to avoid bloated logs.")
	disableTLS := cmd.Flags().Bool("disable-tls", false, "Disables TLS for the local endpoint ONLY. All traffic is still sent, via TLS, to Azure.")
	enableBinFiles := cmd.Flags().Bool("enable-bin-files", false, "Enables writing out amqpproxy-bin files. These files do NOT redact secrets")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		slogger := logging.SloggerFromContext(ctx)

		cf, err := internal.ExtractCommonFlags(cmd)

		if err != nil {
			return err
		}

		slogger.Info("Connecting", "host", cf.Host)

		var baseBinName string

		if *enableBinFiles {
			baseBinName = filepath.Join(cf.LogsDir, "amqpproxy-bin")
		}

		fi, err := amqpproxy.NewAMQPProxy(
			"localhost:5671",
			cf.Host,
			&amqpproxy.AMQPProxyOptions{
				BaseJSONName:               filepath.Join(cf.LogsDir, "amqpproxy-traffic"),
				TLSKeyLogFile:              filepath.Join(cf.LogsDir, "amqpproxy-tlskeys.txt"),
				BaseBinName:                baseBinName,
				DisableTLSForLocalEndpoint: *disableTLS,
				DisableStateTracing:        *disableStateTracking,
				CertDir:                    cf.CertDir,
				ExcludePayloadData:         *excludePayloadData,
			})

		if err != nil {
			return err
		}

		go func() {
			<-ctx.Done()

			slogger.Info("Cancellation received, closing AMQP proxy")

			if err := fi.Close(); err != nil {
				slogger.Error("failed when closing the AMQP Proxy", "error", err)
			}
		}()

		return fi.ListenAndServe()
	}

	return cmd
}
