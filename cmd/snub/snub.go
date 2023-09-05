package main

import (
	"fmt"

	"nostrocket/subrocketLibraries/snub"
)

func main() {
	rootCmd := snub.RootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
