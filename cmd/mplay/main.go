// cmd/mplay/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := newRootCmd().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	return newRootCmdWithFactories(defaultAppFactories())
}

func newRootCmdWithFactories(factories appFactories) *cobra.Command {
	factories = factories.withDefaults()

	cmd := &cobra.Command{
		Use:   "mplay [url]",
		Short: "A unified music player for Spotify and YouTube",
		Long:  "Play songs from supported music links. YouTube playback is implemented; Spotify track previews play when credentials and preview URLs are available.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPlay(cmd, args, factories.playback)
		},
	}

	cmd.AddCommand(
		newQueueCmd(factories),
		newPauseCmd(factories.control),
		newResumeCmd(factories.control),
		newNextCmd(factories.control),
		newStopCmd(factories.control),
		newStatusCmd(factories.control),
		newAuthCmd(factories.config),
	)

	return cmd
}

func runPlay(cmd *cobra.Command, args []string, newPlayback appFactory) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	application, err := newPlayback()
	if err != nil {
		return err
	}
	defer application.Close()

	return application.PlayURL(cmd.Context(), args[0])
}
