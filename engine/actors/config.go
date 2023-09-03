package actors

import (
	"os"

	"github.com/spf13/viper"
	"nostrocket/engine/library"
)

// Ignition
const IgnitionEvent string = "1bf16cac62588cfd7e3c336b8548fa49a09627f03dbf06c7a4fee27bc01972c8"
const IgnitionAccount library.Account = "546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882"
const IgnitionRocketID library.Sha256 = "041b8698b3b7206feca17c89f3f861c31d82dc2fdf3f1d0f25c3bddfa68c64e2"

// Anchors for incoming event trees
const StateChangeRequests string = "120205879a8d9a38adcb794f7cbff3872c4117a7bb7e86672484f6dee7d6b1c6"
const Identity string = "ae14dd661475351993f626f66df8052ed73166796e5cd893c09e4d333e170bb5" //"320c1d0a15bd0d84c3527862ad02d558df3893dfbbc488dcf7530abec25d23bb"
const Merits string = "9f7211ac022b500a7adeeacbe44bb84225d1bb1ee94169f8c5d8d1640a154cbc"   //"083e612017800c276fbbeda8fe3a965daf63bb3030dd0535cfcd7d06afabb870"
const Rockets string = "0f56599b6530f1ed1c11745b76a0d0fc29934e9a90accce1521f4dfac7a78532"
const Problems string = "edea7c22992a1001de805f690d6198fd365ec45e7e5444482100e22447c657a0" //77c3bf5382b62d16a70df8e2932a512e2fce72458ee47b73feaef8ae8b9bd62b
const Problems1 string = "37993b56525f84b814b372acfb69c4474951ee255a104ae6fbb2182623ed7ac1"

// Anchors for outgoing event trees
const CurrentStates string = "697190ea359f0aa00e9e95d8837ab7553764084ca019d9c85089edb04fdc8966" //"fc54dcb214e86ed3049aec2e26199b457866989da0d9acb2bf8313e023344052"

// Anchors for bidirectional event trees
const ReplayPrevention string = "9ab11d92bdeffd28762374d5dfc5286e0f494d7cff5bc00b4fce177bf1115b94" //"e29992d4c7d272dfc274b8a68f735c76dd361a24cc08bdf2ed6fe8808485024c"
const ConsensusTree string = "30cd010a0e79769feb545ffae7820333069894105e063acb50f8315c213f8293"    //"0e4eb74ff5031663115958e66ba1538cd4eadaf91f6599c0b0795e6b4c7bc9af"

// InitConfig sets up our Viper config object
func InitConfig(config *viper.Viper) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		LogCLI(err.Error(), 0)
	}
	config.SetDefault("rootDir", homeDir+"/nostrocket/")
	config.SetConfigType("yaml")
	config.SetConfigFile(config.GetString("rootDir") + "config.yaml")
	err = config.ReadInConfig()
	if err != nil {
		LogCLI(err.Error(), 4)
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
		LogCLI(err.Error(), 0)
	}
}

func initRootDir(conf *viper.Viper) {
	if err := CreateDirectoryIfNotExists(conf.GetString("rootDir")); err != nil {
		LogCLI(err, 0)
	}
}

func CreateDirectoryIfNotExists(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(path, 0755)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil
}

var conf *viper.Viper

func MakeOrGetConfig() *viper.Viper {
	return conf
}

func SetConfig(config *viper.Viper) {
	conf = config
}
