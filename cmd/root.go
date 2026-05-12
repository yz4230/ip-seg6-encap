package cmd

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "ip-seg6-encap",
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		handler := tint.NewHandler(os.Stderr, &tint.Options{
			AddSource:  true,
			TimeFormat: time.TimeOnly,
			Level:      slog.LevelDebug,
		})
		slog.SetDefault(slog.New(handler))
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}
