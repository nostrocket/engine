package snub

import (
	"fmt"

	"github.com/spf13/viper"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func (r *Repo) initConfig() error {
	r.Config = viper.New()
	if fmt.Sprintf("%c", r.Anchor.LocalDir[len(r.Anchor.LocalDir)-1]) != "/" {
		r.Anchor.LocalDir = r.Anchor.LocalDir + "/"
	}
	r.Config.SetDefault("repoPath", r.Anchor.LocalDir)
	r.Config.SetDefault("snubPath", r.Config.GetString("repoPath")+".snub/")
	err := actors.CreateDirectoryIfNotExists(r.Config.GetString("snubPath"))
	if err != nil {
		return err
	}
	r.Config.SetConfigType("yaml")
	r.Config.SetConfigFile(r.Config.GetString("snubPath") + "config.yaml")
	err = r.Config.ReadInConfig()
	if err != nil {
		actors.LogCLI(err.Error(), 4)
	}
	r.Config.SetDefault("firstRun", true)
	r.Config.SetDefault("dTag", r.Anchor.DTag)
	r.Config.SetDefault("relays", []string{"ws://127.0.0.1:8080"})
	r.Config.SetDefault("PoW", 0)
	err = library.Touch(r.Config.GetString("snubPath") + "config.yaml")
	if err != nil {
		return err
	}
	err = r.Config.WriteConfig()
	if err != nil {
		return err
	}
	return nil
}
