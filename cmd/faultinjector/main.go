package main

import (
	"context"
	"log/slog"
)

func main() {
	rootCmd := newRootCommand()

	// disconnect commands
	rootCmd.AddCommand(newDisconnectCommand(context.Background()))

	// detach commands
	rootCmd.AddCommand(newDetachAfterDelayCommand(context.Background()))
	rootCmd.AddCommand(newDetachAfterTransferCommand(context.Background()))

	// transfer commands
	rootCmd.AddCommand(newSlowTransferFrames(context.Background()))
	rootCmd.AddCommand(newMultiTransferInjector(context.Background()))

	// passthrough/diagnostics
	rootCmd.AddCommand(newPassthroughCommand(context.Background()))

	if err := rootCmd.Execute(); err != nil {
		slog.Error("Failed to run command", "error", err)
	}
}
