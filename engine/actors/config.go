package actors

import (
	"os"

	"github.com/spf13/viper"
	"nostrocket/engine/library"
)

// InitConfig sets up our Viper config object
func InitConfig(config *viper.Viper) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		library.LogCLI(err.Error(), 0)
	}
	config.SetDefault("rootDir", homeDir+"/nostrocket/")
	config.SetConfigType("yaml")
	config.SetConfigFile(config.GetString("rootDir") + "config.yaml")
	err = config.ReadInConfig()
	if err != nil {
		library.LogCLI(err.Error(), 4)
	}
	config.SetDefault("firstRun", true)
	config.SetDefault("flatFileDir", "data/")
	config.SetDefault("blockServer", "https://blockchain.info")
	config.SetDefault("logLevel", 4)
	config.SetDefault("doNotPublish", false)
	config.SetDefault("ignitionHeight", int64(761151))
	config.SetDefault("websocketAddr", "0.0.0.0:1031")
	config.SetDefault("fastSync", true)

	//we usually lean towards errors being fatal to cause less damage to state. If this is set to true, we lean towards staying alive instead.
	config.SetDefault("highly_reliable", false)
	config.SetDefault("forceBlocks", false)
	config.SetDefault("relaysMust", []string{"wss://15171031.688.org"})
	// Create our working directory and config file if not exist
	initRootDir(config)
	library.Touch(config.GetString("rootDir") + "config.yaml")
	err = config.WriteConfig()
	if err != nil {
		library.LogCLI(err.Error(), 0)
	}
}

func initRootDir(conf *viper.Viper) {
	_, err := os.Stat(conf.GetString("rootDir"))
	if os.IsNotExist(err) {
		err = os.Mkdir(conf.GetString("rootDir"), 0755)
		if err != nil {
			library.LogCLI(err, 0)
		}
	}
}

var conf *viper.Viper

func MakeOrGetConfig() *viper.Viper {
	return conf
}

func SetConfig(config *viper.Viper) {
	conf = config
}
