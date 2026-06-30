// cmd/mplay/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lucasansei/multiplat-playlist/internal/app"
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
	cmd := &cobra.Command{
		Use:   "mplay [url]",
		Short: "A unified music player for Spotify and YouTube",
		Long:  "Play songs from supported music links. YouTube playback is implemented; Spotify playback is planned.",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runPlay,
	}

	cmd.AddCommand(
		newQueueCmd(),
		newPauseCmd(),
		newResumeCmd(),
		newNextCmd(),
		newStopCmd(),
		newStatusCmd(),
		newAuthCmd(),
	)

	return cmd
}

func runPlay(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	application, err := app.NewPlayback()
	if err != nil {
		return err
	}
	defer application.Close()

	return application.PlayURL(cmd.Context(), args[0])
}
