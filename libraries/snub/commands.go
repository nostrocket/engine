package snub

import (
	"fmt"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/cobra"
	"nostrocket/messaging/relays"
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

	init := &cobra.Command{
		Use:   "init",
		Short: "add a snub config file to the repository",
		Run: func(cmd *cobra.Command, args []string) {
			LoadRocketConfig()
			BuildFromExistingRepo(NewRepoOptions{
				Path: GetCurrentDirectory(),
			})
		},
	}
	rootCmd.AddCommand(init)

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

	example := &cobra.Command{
		Use:   "example",
		Short: "pass in an a tag to print some example events",
		Run: func(cmd *cobra.Command, args []string) {
			sampleEvents(flagValue)
		},
	}
	example.Flags().StringVarP(&flagValue, "atag", "a", "", "an a tag in the format 31228:<pubkey>:<d tag>")
	rootCmd.AddCommand(example)

	return rootCmd
}

func sampleEvents(atag string) {
	//31228:546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882:1d16e64ebcf5cf13640c53c6e1a341b5ee3b868efea110ecef581c98d4dfe023]
	events := make(map[int64]nostr.Event)
	tm := make(nostr.TagMap)
	tm["a"] = []string{atag}
	tm2 := make(nostr.TagMap)
	d := strings.Split(atag, ":")
	tm2["d"] = []string{d[2]}
	n := relays.FetchEvents([]string{"ws://127.0.0.1:8080"}, nostr.Filters{
		nostr.Filter{Tags: tm},
		nostr.Filter{Tags: tm2},
	})
	for _, event := range n {
		events[int64(event.Kind)] = event
	}
	for _, event := range events {
		switch event.Kind {
		case 3121:
			fmt.Printf("\n-----THIS IS A COMMIT EVENT-----\n%#v\n\n", event)
		case 3123:
			fmt.Printf("\n-----THIS IS A BLOB EVENT-----\nBinary blobs are compressed and hex encoded to make events smaller, you can see this data in the data tag.\n%#v\n\n", event)
		case 3122:
			fmt.Printf("\n-----THIS IS A TREE EVENT-----\n%#v\n\n", event)
		case 31227:
			fmt.Printf("\n-----THIS IS A BRANCH EVENT-----\nReplaeable event so that the HEAD can be udpated. The list of commits is not strictly neccessary and will probably be removed\n%#v\n\n", event)
		case 31228:
			fmt.Printf("\n-----THIS IS A REPOSITORY ANCHOR EVENT-----\nThis is a replaceable event so that maintainers etc can be updated\n%#v\n\n", event)
		}
	}
}
