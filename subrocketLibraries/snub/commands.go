package snub

import (
	"fmt"

	"github.com/spf13/cobra"
)

func RootCommand() *cobra.Command {
	var flagValue string
	rootCmd := &cobra.Command{
		Use:   "snub",
		Short: "Welcome to snub. I didn't choose the snub life; the snub life chose me.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				return
			}

		},
	}

	// Add a flag to the root command
	rootCmd.Flags().StringVarP(&flagValue, "force", "f", "", "A flag value")

	// Add a subcommand to the root command
	clone := &cobra.Command{
		Use:   "clone",
		Short: "clone a remote repository from a url or nostr event ID in hex or bech32 format",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Flag value: %s\n", flagValue)
		},
	}
	clone.Flags().StringVarP(&flagValue, "nostr", "n", "", "A nostr event ID in hex or note format")
	rootCmd.AddCommand(clone)

	var name string
	publish := &cobra.Command{
		Use:   "publish",
		Short: "publish this repository as a new canonical remote that lives natively on nostr",
		Long:  "This command publishes the current repository as a series of events containing git objects.",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			simulate, err := cmd.Flags().GetBool("simulate")
			if err != nil {
				return err
			}
			return Publish(NewRepoOptions{
				Name:         name,
				UpstreamDTag: "", //todo
				ForkedAt:     "", //todo
				Simulate:     simulate,
			})
		},
	}
	publish.Flags().StringVarP(&name, "name", "n", "", "Publish this repository as the provided name instead of using the current directory name")
	publish.Flags().BoolP("simulate", "s", false, "Don't actually publish anything, just simulate a print the events that would be published")
	rootCmd.AddCommand(publish)

	push := &cobra.Command{
		Use:   "push",
		Short: "push local changes to the remote repository",
		Long:  "This command pushes the currently checked out branch to the remote repository.",
		//Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			repository, err := openRepository(GetCurrentDirectory())
			if err != nil {
				panic(err)
			}
			fmt.Println(repository.Head())
		},
	}
	rootCmd.AddCommand(push)

	remote := &cobra.Command{
		Use:   "addremote",
		Short: "add a remote to this repository, must be a valid nostr event of kind 31228",
		Long: "This command adds a remote to this repository. It must be a kind 31228 event, identified by its ID\n" +
			"in hex or bech32 (note1...)",
		Run: func(cmd *cobra.Command, args []string) {
			//fmt.Printf("Flag value: %s\n", flagValue)
		},
	}
	rootCmd.AddCommand(remote)

	return rootCmd
}
