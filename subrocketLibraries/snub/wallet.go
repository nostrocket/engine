package snub

import (
	"fmt"

	"github.com/spf13/viper"
	"nostrocket/engine/actors"
)

var loaded bool

func LoadRocketConfig() {
	if !loaded {
		loaded = true
		rocketConf := viper.New()
		//Now we initialise this configuration with basic settings that are required on startup.
		actors.InitConfig(rocketConf)
		//make the config accessible globally
		actors.SetConfig(rocketConf)
		fmt.Println("Using pubkey: " + actors.MyWallet().Account)
	}
}
