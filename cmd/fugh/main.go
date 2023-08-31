package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/viper"
	"nostrocket/engine/actors"
	"nostrocket/messaging/relays"
	"nostrocket/subrocketLibraries/fugh"
)

func main() {
	conf := viper.New()
	//Now we initialise this configuration with basic settings that are required on startup.
	actors.InitConfig(conf)
	//make the config accessible globally
	actors.SetConfig(conf)
	fmt.Println("Current pubkey: " + actors.MyWallet().Account)
	r, err := fetchRepo("81af21763dbe36827e89ac2e1757c484238979be126b564002f56022f267b09e")
	if err != nil {
		actors.LogCLI(err, 0)
	}
	fmt.Printf("\n%#v\n", r)
	rEvents, err := r.FetchAllEvents()
	if err != nil {
		panic(err)
	}
	for _, event := range rEvents {
		if event.Kind == 31227 {
			branch, err := fugh.GetBranchFromEvent(event)
			if err != nil {
				panic(err)
			}
			fmt.Printf("\n%#v\n", branch)
		}
	}
	//e, err := r.CreateRepoEvent()
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Printf("\n%#v\n", e)
	//sender := actors.StartRelaysForPublishing([]string{"wss://nostr.688.org"})
	//sender <- e
	//time.Sleep(time.Second * 10)

	//repoPath := "/Users/gareth/git/nostrocket/test/fugh"
	//treeSHA, err := fugh.GetFirstTreeSHA(repoPath)
	//if err != nil {
	//	actors.LogCLI(err, 0)
	//}
	//
	////fmt.Println("First tree SHA:", treeSHA)
	//
	//identifiers, err := fugh.GetBlobIdentifiers(repoPath, treeSHA)
	//if err != nil {
	//	actors.LogCLI(err, 1)
	//}
	//for _, identifier := range identifiers {
	//	fmt.Println(identifier)
	//}
	//
	//blob, err := fugh.GetBinaryBlob(repoPath, identifiers[0])
	//if err != nil {
	//	actors.LogCLI(err, 1)
	//}
	//fmt.Printf("40: %s", blob)
	//
	//sha1Hash := sha1.Sum(blob)
	//sha1HashString := hex.EncodeToString(sha1Hash[:])
	//
	//fmt.Printf("SHA1: %s\n", sha1HashString)
	//
	//blobMap, err := fugh.CreateBlobMap(repoPath)
	//if err != nil {
	//	actors.LogCLI(err, 1)
	//}
	//var saved int
	//var total int
	//for s, b := range blobMap {
	//	c, err := compressBytes(b)
	//	if err != nil {
	//		actors.LogCLI(err, 1)
	//	}
	//	total += len(b)
	//	saved += len(b) - len(c)
	//	fmt.Printf("blob ID: %s\n%x\n\n", s, c)
	//}
	//
	////repo := fugh.Repo{}
	////repo.Maintainers = append(repo.Maintainers, actors.MyWallet().Account)
	////repo.DTag = library.Random()
	////repo.Name = "testing"
	////e, err := repo.CreateRepoEvent()
	////if err != nil {
	////	actors.LogCLI(err, 0)
	////}
	////fmt.Printf("\n%#v\n", e)
	////sender := actors.StartRelaysForPublishing([]string{"wss://nostr.688.org"})
	////sender <- e
	//branch := fugh.Branch{
	//	Name:    "master",
	//	Head:    "",
	//	Root:    "4c467d089f8a4f3c0bce308a3ddc6f3bcc376aaeecd6991af0b449ffa79e0e3a",
	//	ATag:    "31228:546b4d7f86fe2c1fcc7eb10bf96c2eaef1daa26c67dad348ff0e9c853ffe8882:81af21763dbe36827e89ac2e1757c484238979be126b564002f56022f267b09e",
	//	DTag:    library.Random(),
	//	Commits: nil,
	//	Length:  0,
	//}
	//b, err := branch.CreateBranchEvent(&fugh.Repo{
	//	Anchor:      "4c467d089f8a4f3c0bce308a3ddc6f3bcc376aaeecd6991af0b449ffa79e0e3a",
	//	Maintainers: []string{actors.MyWallet().Account},
	//})
	//if err != nil {
	//	actors.LogCLI(err, 0)
	//}
	//fmt.Printf("\n%#v\n", b)
	//sender := actors.StartRelaysForPublishing([]string{"wss://nostr.688.org"})
	//sender <- b
	//time.Sleep(time.Second * 10)
}

func fetchRepo(repoDTag string) (fugh.Repo, error) {
	tm := make(nostr.TagMap)
	tm["d"] = []string{repoDTag}
	n := relays.FetchEvents([]string{"wss://nostr.688.org"}, nostr.Filters{nostr.Filter{
		Kinds: []int{31228},
		Tags:  tm,
		//IDs: []string{repoID},
	}})
	if len(n) > 0 {
		fmt.Printf("\n%#v\n", n[0])
		r, err := fugh.GetRepoFromEvent(n[0])
		if err != nil {
			return fugh.Repo{}, err
		}
		return r, nil
	}
	return fugh.Repo{}, fmt.Errorf("could not find repo event")
}

func compressBytes(input []byte) ([]byte, error) {
	var compressed bytes.Buffer
	gzWriter := gzip.NewWriter(&compressed)
	_, err := gzWriter.Write(input)
	if err != nil {
		return nil, fmt.Errorf("failed to compress string: %v", err)
	}

	err = gzWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %v", err)
	}

	return compressed.Bytes(), nil
}

//
//func main() {
//	repoURL := "https://github.com/nostrocket/flamebucketmanager.git"
//	localRepoPath := "."
//
//	err := fetchChanges(repoURL, localRepoPath)
//	if err != nil {
//		fmt.Printf("Failed to fetch changes: %s\n", err.Error())
//		return
//	}
//
//	fmt.Println("Changes fetched successfully!")
//}

func getGitReference(repoURL string) (string, error) {
	referenceURL := repoURL + "/info/refs?service=git-upload-pack"

	resp, err := http.Get(referenceURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	reference := string(body)
	reference = reference[4:] // Remove the "001e#" prefix

	return reference, nil
}

func downloadObjects(repoURL, localRepoPath, reference string) error {
	packfileURL := repoURL + "/objects/pack/pack-" + reference + ".pack"

	resp, err := http.Get(packfileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	packfilePath := filepath.Join(localRepoPath, "packs", "pack-"+reference+".pack")
	packfile, err := os.Create(packfilePath)
	if err != nil {
		return err
	}
	defer packfile.Close()

	_, err = io.Copy(packfile, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func fetchChanges(repoURL, localRepoPath string) error {
	// Step 1: Fetch the Git reference
	reference, err := getGitReference(repoURL)
	if err != nil {
		return err
	}

	// Step 2: Download the objects
	err = downloadObjects(repoURL, localRepoPath, reference)
	if err != nil {
		return err
	}

	return nil
}
