package main

import (
	"fmt"

	"github.com/spf13/viper"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func main() {
	// Various aspect of this application require global and local settings. To keep things
	// clean and tidy we put these settings in a Viper configuration.
	conf := viper.New()

	// Now we initialise this configuration with basic settings that are required on startup.
	actors.InitConfig(conf)
	// make the config accessible globally
	actors.SetConfig(conf)
	fmt.Println("CURRENT CONFIG")
	for k, v := range actors.MakeOrGetConfig().AllSettings() {
		fmt.Printf("\nKey: %s; Value: %v\n", k, v)
	}
	fmt.Println(library.Bye())
}
