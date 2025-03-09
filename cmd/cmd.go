package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/s0lesurviv0r/gist-backup/github"
)

func Run() {
	var (
		debug    bool
		username string
		dst      string
		token    string
	)

	cmd := &cobra.Command{
		Use:   "gist-backup",
		Short: "Backup your GitHub Gists",
		Long:  "Backup your GitHub Gists",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.SetOutput(os.Stdout)
			log.SetFormatter(&log.TextFormatter{
				FullTimestamp: true,
				ForceColors:   true,
			})
			if debug {
				log.SetLevel(log.DebugLevel)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()

			client := github.NewClient(token, nil)

			log.Info("Starting backup...")

			err := client.DownloadAllGistsForUser(ctx, username, dst)
			if err != nil {
				log.Fatalf("Error downloading gists: %v", err)
			}

			log.Info("Backup completed")
		},
	}

	cmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")

	cmd.Flags().StringVarP(&username, "username", "u", "", "GitHub username")
	cmd.Flags().StringVarP(&dst, "dst", "d", "", "Target directory")
	cmd.Flags().StringVarP(&token, "token", "k", "", "GitHub token")

	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("dst")

	cmd.Execute()
}
