package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/eiannone/keyboard"
	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/viper"
	"nostrocket/engine/actors"
	"nostrocket/messaging/blocks"
)

func main() {
	conf := viper.New()
	//Now we initialise this configuration with basic settings that are required on startup.
	actors.InitConfig(conf)
	//make the config accessible globally
	actors.SetConfig(conf)
	fmt.Println("Current wallet: " + actors.MyWallet().Account)
	var eventChan = make(chan nostr.Event)
	var terminate = make(chan struct{})
	go sendBlocks(eventChan, terminate)
	go listenForBlocks(eventChan, terminate)
	go cliListener(terminate)
	<-terminate
}

func cliListener(interrupt chan struct{}) {
	for {
		r, k, err := keyboard.GetSingleKey()
		if err != nil {
			panic(err)
		}
		str := string(r)
		switch str {
		default:
			if k == 13 {
				fmt.Println("\n-----------------------------------")
				break
			}
			if r == 0 {
				break
			}
			fmt.Println("Key " + str + " is not bound to any test procedures. See main.cliListener for more details.")
		case "q":
			close(interrupt)
		}
	}
}

func listenForBlocks(eventChan chan nostr.Event, terminate chan struct{}) {
	var currentHeight = checkAndSend(0, eventChan)
	for {
		select {
		case <-terminate:
			return
		case <-time.After(time.Second * 30):
			currentHeight = checkAndSend(currentHeight, eventChan)
		}
	}
}

func checkAndSend(curentHeight int64, eventChan chan nostr.Event) int64 {
	block, err := getLatestBlock()
	if err != nil {
		actors.LogCLI(err, 3)
	}
	if block.Height > curentHeight {
		eventChan <- makeEvent(block)
		fmt.Printf("\n%#v\n", block)
		fmt.Printf("\n%#v\n", makeEvent(block))
		return block.Height
	}
	return curentHeight
}

func sendBlocks(eventChan chan nostr.Event, terminate chan struct{}) {
	for {
		select {
		case <-terminate:
			return
		case e := <-eventChan:
			e.ID = e.GetID()
			e.Sign(actors.MyWallet().PrivateKey)
			//e.Sign("")
			fmt.Printf("\n%#v\n%d\n", e, e.CreatedAt)

			sigok, err := e.CheckSignature()
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			if !sigok {
				fmt.Println("sig failed")
				return
			}
			if sigok {
				fmt.Println("sig ok")
				sendChan := make(chan nostr.Event)
				startRelays(sendChan)
				sendChan <- e
				fmt.Printf("%#v", e)
				time.Sleep(time.Millisecond * 500)
			}
		}

	}
}

func makeEvent(block blocks.Block) (n nostr.Event) {
	n.PubKey = actors.MyWallet().Account
	n.Kind = 1517
	n.Content = ""
	n.CreatedAt = nostr.Timestamp(time.Now().Unix())
	tags := nostr.Tags{}
	tags = append(tags, nostr.Tag{"hash", block.Hash})
	tags = append(tags, nostr.Tag{"height", fmt.Sprintf("%d", block.Height)})
	tags = append(tags, nostr.Tag{"difficulty", fmt.Sprintf("%d", block.Difficulty)})
	tags = append(tags, nostr.Tag{"minertime", fmt.Sprintf("%d", block.MinerTime.Unix())})
	tags = append(tags, nostr.Tag{"mediantime", fmt.Sprintf("%d", block.MedianTime.Unix())})
	n.Tags = tags
	return
}

func getLatestBlock() (rb blocks.Block, e error) {
	hash, err := getHash()
	if err != nil {
		actors.LogCLI(err, 3)
	}
	block, err := getBlock(hash)
	if err != nil {
		actors.LogCLI(err, 3)
	}
	return block, nil
}

func getBlock(hash string) (block blocks.Block, e error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://blockstream.info/api/block/"+hash, nil)
	if err != nil {
		return block, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return block, err
	}
	if resp.StatusCode != 200 {
		return block, fmt.Errorf("http response error code %d", resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return block, err
	}
	var responseObject BlockStream
	err = json.Unmarshal(bodyBytes, &responseObject)
	if err != nil {
		spew.Dump(bodyBytes)
		return block, err
	}
	block.MinerTime = time.Unix(responseObject.Timestamp, 0)
	block.MedianTime = time.Unix(responseObject.Mediantime, 0)
	block.Difficulty = responseObject.Difficulty
	block.Height = responseObject.Height
	block.Hash = responseObject.Id
	return
}

func getHash() (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://blockstream.info/api/blocks/tip/hash", nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		panic(resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if len(fmt.Sprintf("%s", bodyBytes)) != 64 {
		return "", fmt.Errorf("invalid hash")
	}
	return fmt.Sprintf("%s", bodyBytes), nil
}

type BlockStream struct {
	Id                string `json:"id"`
	Height            int64  `json:"height"`
	Version           int64  `json:"version"`
	Timestamp         int64  `json:"timestamp"`
	TxCount           int64  `json:"tx_count"`
	Size              int64  `json:"size"`
	Weight            int64  `json:"weight"`
	MerkleRoot        string `json:"merkle_root"`
	Previousblockhash string `json:"previousblockhash"`
	Mediantime        int64  `json:"mediantime"`
	Nonce             int64  `json:"nonce"`
	Bits              int64  `json:"bits"`
	Difficulty        int64  `json:"difficulty"`
}

func startRelays(sendChan chan nostr.Event) {
	relay, err := nostr.RelayConnect(context.Background(), "wss://nostr.688.org") //"ws://127.0.0.1:45321") //"wss://nostr.688.org")
	if err != nil {
		panic(err)
	}

	go func() {
		select {
		case e := <-sendChan:
			_, err := relay.Publish(context.Background(), e)
			if err != nil {
				actors.LogCLI(err.Error(), 2)
			}
		}
	}()
}
