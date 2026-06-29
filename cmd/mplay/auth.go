package main

import (
	"github.com/lucasansei/multiplat-playlist/internal/app"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Spotify",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.NewConfig()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.AuthSpotify()
		},
	}
}
