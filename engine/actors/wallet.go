package actors

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/nbd-wtf/go-nostr/nip06"
	"github.com/sasha-s/go-deadlock"
	"nostrocket/engine/library"
)

var currentWallet library.Wallet
var currentWalletMutex = &deadlock.Mutex{}

// MyWallet returns the current Wallet or creates a new one if there isn't one already
func MyWallet() library.Wallet {
	currentWalletMutex.Lock()
	defer currentWalletMutex.Unlock()
	if len(currentWallet.PrivateKey) == 0 {
		//try to restore wallet from disk
		if w, ok := getWalletFromDisk(); ok {
			currentWallet = w
		} else {
			LogCLI("Generating a new wallet, write down the seed words if you want to keep it", 4)
			currentWallet = makeNewWallet()
			fmt.Printf("\n\n~NEW WALLET~\nPublic Key: %s\nPrivate Key: %s\nSeed Words: %s\n\n", currentWallet.Account, currentWallet.PrivateKey, currentWallet.SeedWords)
		}
	}
	if err := persistCurrentWallet(); err != nil {
		LogCLI(err.Error(), 0)
	}
	return currentWallet
}

func makeNewWallet() library.Wallet {
	seedWords, err := nip06.GenerateSeedWords()
	if err != nil {
		LogCLI(err.Error(), 0)
	}
	seed := nip06.SeedFromWords(seedWords)
	sk, err := nip06.PrivateKeyFromSeed(seed)
	if err != nil {
		LogCLI(err.Error(), 0)
	}
	return library.Wallet{
		PrivateKey: sk,
		SeedWords:  seedWords,
		Account:    getPubKey(sk),
	}
}

func getPubKey(privateKey string) string {
	if keyb, err := hex.DecodeString(privateKey); err != nil {
		LogCLI(fmt.Sprintf("Error decoding key from hex: %s\n", err.Error()), 0)
	} else {
		_, pubkey := btcec.PrivKeyFromBytes(keyb)
		return hex.EncodeToString(pubkey.X().Bytes())
	}
	return ""
}

func persistCurrentWallet() error {
	file, err := os.Create(MakeOrGetConfig().GetString("rootDir") + "wallet.dat")
	if err != nil {
		LogCLI(err.Error(), 0)
	}
	defer file.Close()
	bytes, err := json.Marshal(currentWallet)
	if err != nil {
		LogCLI(err.Error(), 0)
	}
	_, err = file.Write(bytes)
	if err != nil {
		LogCLI(err.Error(), 0)
	}
	return nil
}

func getWalletFromDisk() (w library.Wallet, ok bool) {
	file, err := ioutil.ReadFile(MakeOrGetConfig().GetString("rootDir") + "wallet.dat")
	if err != nil {
		LogCLI(fmt.Sprintf("Error getting wallet file: %s", err.Error()), 2)
		return library.Wallet{}, false
	}
	err = json.Unmarshal(file, &w)
	if err != nil {
		LogCLI(fmt.Sprintf("Error parsing wallet file: %s", err.Error()), 3)
		return library.Wallet{}, false
	}
	return w, true
}
