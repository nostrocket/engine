package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sasha-s/go-deadlock"
	"github.com/spf13/viper"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
	"nostrocket/messaging/eventconductor"
)

func main() {
	deadlock.Opts.DeadlockTimeout = time.Second * 5
	// Various aspect of this application require global and local settings. To keep things
	// clean and tidy we put these settings in a Viper configuration.
	conf := viper.New()

	// Now we initialise this configuration with basic settings that are required on startup.
	actors.InitConfig(conf)
	// make the config accessible globally
	actors.SetConfig(conf)
	printArt()
	terminateChan := make(chan struct{})
	actors.SetTerminateChan(terminateChan)
	if os.Getenv("NOSTROCKET_DEBUG") == "true" {
		go func() {
			select {
			case <-time.After(time.Second * 30):
				close(terminateChan)
			}
		}()
	} else {
		go cliListener(terminateChan)
	}
	wg := &deadlock.WaitGroup{}
	actors.SetWaitGroup(wg)
	eventconductor.Start()
	<-terminateChan
	actors.MakeOrGetConfig().Set("firstRun", false)
	err := actors.MakeOrGetConfig().WriteConfig()
	if err != nil {
		library.LogCLI(err.Error(), 3)
	}
	wg.Wait()
	//todo use waitgroup
	fmt.Println(library.Bye())
}

func printArt() {
	if actors.MakeOrGetConfig().GetBool("firstRun") {
		//Just giving everyone a chance to soak this up on the first run
		scanner := bufio.NewScanner(strings.NewReader(library.Hello()))
		for scanner.Scan() {
			time.Sleep(time.Millisecond * 127)
			fmt.Println(scanner.Text())
		}
		fmt.Println()
		time.Sleep(time.Second)
	} else {
		fmt.Printf("\n%s\n", library.Hello())
	}
	//507ms of silence in memory of those who have used branle
	time.Sleep(time.Millisecond * 507)
}
