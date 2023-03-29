package actors

import (
	"os"

	"github.com/spf13/viper"
	"nostrocket/engine/library"
)

const IgnitionEvent string = "fd459ea06157e30cfb87f7062ee3014bc143ecda072dd92ee6ea4315a6d2df1c"
const StateChangeRequests string = "7a22f580d253c4142aa4e6b28d577b2d59fdd30083b0eb27ee76a9bd750bff26"
const ReplayPrevention string = "24c30ad7f036ed49379b5d1209836d1ff6795adb34da2d3e4cabc47dc9dfef21"
const Identity string = "0a73208becd0b1a9d294e6caef14352047ab44b848930e6979937fe09effaf71"
const CurrentStates string = "0255594820a3ddc5b603d4e37ba6b2325879aebec401b86f9d69f5fd3864c203"
const IgnitionAccount library.Account = "b4f36e2a63792324a92f3b7d973fcc33eaa7720aaeee71729ac74d7ba7677675"

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
	config.SetDefault("relaysMust", []string{"wss://nostr.688.org"})
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
