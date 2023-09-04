package snub

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/viper"
	"nostrocket/engine/library"
)

type Repo struct {
	Anchor   RepoAnchor
	Commits  map[library.Sha1]Commit
	Trees    map[library.Sha1]Commit
	Branches map[string]Branch //[<branch name>]Branch
	Blobs    map[library.Sha1]Blob
	Git      git.Repository
	Config   *viper.Viper
}

func (r *Repo) loadFromDisk() error {
	repository, err := openRepository(r.Anchor.LocalDir)
	if err != nil {
		return err
	}
	r.Git = *repository
	return nil
}

func (r *Repo) Init() error {
	err := r.loadFromDisk()
	if err != nil {
		return err
	}
	err = r.initConfig()
	if err != nil {
		return err
	}
	return nil
}

type Commit struct {
	//how do we get commits from events and parse them into git objects? hash-object?
	//git cat-file -p df5af114df19730dc1d2936e5819e07273182a76  | git hash-object -t commit --stdin
	GID          library.Sha1
	Author       LegacyIdentification //used for legacy operations when pulling from git repos, we don't need author, in snub when merging two or more parents we can see who the authors are from the parents
	Committer    LegacyIdentification //the person who publishes the event
	Message      string
	ParentIDs    []library.Sha1
	TreeID       library.Sha1
	EventID      library.Sha256
	LegacySig    string
	LegacyBackup string //the raw text of the commit from existing repo that uses emails as ID and GPG sigs
}

func (c *Commit) String() (s string) {
	s += "tree " + c.TreeID + "\n"
	for _, d := range c.ParentIDs {
		s += "parent " + d + "\n"
	}
	s += c.Author.string() + "\n"
	s += c.Committer.string() + "\n"
	if len(c.LegacySig) > 0 {
		s += "gpgsig " + c.LegacySig
	}
	s += "\n"
	s += c.Message
	return
}

type LegacyIdentification struct {
	Name      string
	Email     string
	Timestamp int64
	UTCoffset string
	Type      string //author | committer
}

func (l *LegacyIdentification) string() string {
	return fmt.Sprintf("%s %s <%s> %d %s", l.Type, l.Name, l.Email, l.Timestamp, l.UTCoffset)
}

func (l *LegacyIdentification) tag() (t nostr.Tag) {
	t = append(t, l.Type, l.Name, l.Email, fmt.Sprintf("%d", l.Timestamp), l.UTCoffset)
	return
}

type Tree struct {
	//use mktree instead of hash-object
	Items   []treeObject //the items in the tree
	GID     library.Sha1
	EventID library.Sha256
}

type treeObject struct {
	name     string       //file name
	gID      library.Sha1 //Git ID of the object
	fileMode int64
	blob     Blob //MUST include a Blob XOR Tree
	tree     Tree //MUST include a Blob XOR Tree
}

type Branch struct {
	Name       string           //name of this branch in plaintext, MUST not contain spaces
	Head       library.Sha1     //commit identifier
	ATag       string           //the part of the "a" tag that points to the repo anchor 31228:<pubkey of repo creator>:<repo event d tag>
	DTag       library.Sha256   //random hash
	Commits    []library.Sha256 //a list of event IDs for all merged commits for convenience when fetching events
	Length     int64            //the total number of commits in this branch
	LastUpdate int64            //timestamp of latest update
}

type RepoAnchor struct {
	CreatedBy    library.Account
	Name         string
	DTag         library.Sha256    //random hash
	UpstreamDTag string            //the upstream repository, only used if this is a fork, format MUST be 31228:<pubkey>:<DTag>
	ForkedAt     library.Sha1      //the commit this was forked at
	Maintainers  []library.Account //only used if NOT integrated with nostrocket
	Rocket       library.RocketID  //only used if integrated with nostrocket to get maintainers etc from there
	LastUpdate   int64             //timestamp of latest update
	LocalDir     string            //the location on the local filesystem if this exists locally
}

type Blob struct {
	GID      library.Sha1
	BlobData []byte
	EventID  library.Sha256
}

type BlobMap map[library.Sha1]Blob
