package main

import "github.com/spf13/cobra"

func newAuthCmd(newConfig appFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Spotify",
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := newConfig()
			if err != nil {
				return err
			}
			defer application.Close()

			return application.AuthSpotify()
		},
	}
}
