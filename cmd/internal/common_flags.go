package internal

import "github.com/spf13/cobra"

const HostFlagName = "host"
const LogsFlagName = "logs"
const CertFlagName = "cert"

type CommonFlags struct {
	Host    string
	LogsDir string
	CertDir string
}

func AddCommonFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(HostFlagName, "", "The hostname of the service we're proxying to (ex: <server>.servicebus.windows.net)")
	cmd.PersistentFlags().String(LogsFlagName, ".", "The directory to write any logs or trace files")
	cmd.PersistentFlags().String(CertFlagName, ".", "The directory to write the TLS server.cert and server.key used for the proxy's endpoint. If the files already exist, they are re-used.")

	_ = cmd.MarkPersistentFlagRequired(HostFlagName)
}

func ExtractCommonFlags(cmd *cobra.Command) (CommonFlags, error) {
	host, err := cmd.Flags().GetString(HostFlagName)

	if err != nil {
		return CommonFlags{}, err
	}

	logs, err := cmd.Flags().GetString(LogsFlagName)

	if err != nil {
		return CommonFlags{}, err
	}

	cert, err := cmd.Flags().GetString(CertFlagName)

	if err != nil {
		return CommonFlags{}, err
	}

	return CommonFlags{
		Host:    host,
		LogsDir: logs,
		CertDir: cert,
	}, nil
}
