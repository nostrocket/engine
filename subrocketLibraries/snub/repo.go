package snub

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func BuildFromExistingRepo(options NewRepoOptions) (r Repo, err error) {
	name, err := getRepoName(options.Name, options.Path)
	if err != nil {
		return Repo{}, err
	}
	r.Anchor = RepoAnchor{
		CreatedBy:    actors.MyWallet().Account,
		Name:         name,
		DTag:         library.Random(),
		UpstreamDTag: options.UpstreamDTag, //optional
		ForkedAt:     options.ForkedAt,     //optional
		Maintainers:  []string{actors.MyWallet().Account},
		LocalDir:     options.Path,
	}
	err = r.Init()
	if err != nil {
		return Repo{}, err
	}
	//get current branch name
	branchName, err := GetCurrentBranch(r.Anchor.LocalDir)
	if err != nil {
		return Repo{}, err
	}
	headCommitID, err := GetCurrentHeadCommitID(r.Anchor.LocalDir)
	if err != nil {
		return Repo{}, err
	}
	branch := Branch{
		Name: branchName,
		Head: headCommitID,
		ATag: r.Anchor.childATag(),
		DTag: library.Random(),
	}
	r.Branches = make(map[string]Branch)
	//get commits
	commitIDs, err := getAllCommitIDs(&r.Git, branch.Head)
	if err != nil {
		return Repo{}, err
	}
	r.Commits = make(map[library.Sha1]Commit)
	for _, d := range commitIDs {
		data, err := getCommitData2(d, &r.Git)
		if err != nil {
			return Repo{}, err
		}
		if data.GID != d {
			panic(62)
		}
		//fmt.Printf("\n\n%#v\n", *data)
		//fmt.Println(data.String())
		//fmt.Println(data.GID)
		hash, err := getGitHashForObject(data.String(), "commit")
		if err != nil {
			return Repo{}, err
		}
		if hash != data.GID {
			actors.LogCLI(fmt.Sprintf("could not replicate git identifier, storing the original commit data as compressed binary for object %s", data.GID), 3)
			cbytes, err := getCommitBytes(data.GID)
			if err != nil {
				return Repo{}, err
			}
			compressed, err := compressBytes(cbytes)
			if err != nil {
				return Repo{}, err
			}
			data.LegacyBackup = fmt.Sprintf("%x", compressed)
			//fmt.Println("fail")
			//fmt.Println(hash)
			//fmt.Println(data.GID)
			//ioutil.WriteFile("ourStrings/"+data.GID, []byte(data.String()), 0644)
			//if err != nil {
			//	return Repo{}, err
			//}
			//ioutil.WriteFile("theirStrings/"+data.GID, cbytes, 0644)
		}
		//if plumbing.ComputeHash(1, []byte(data.String())).String() != data.GID {
		//	fmt.Println(len(r.CommitEventIDs))
		//	fmt.Printf("\nGID: %s CALULATED: %s\n", data.GID, plumbing.ComputeHash(1, []byte(data.String())).String())
		//	fmt.Println("---STRING---")
		//	fmt.Println(data.String())
		//	fmt.Println("---ORIGINAL---")
		//	fmt.Println(data.LegacyBackup)
		//	return Repo{}, fmt.Errorf("failed to calculate the correct hash")
		//}
		r.Commits[data.GID] = *data
		if len(branch.CommitEventIDs) == 0 {
			branch.CommitEventIDs = make(map[library.Sha256]library.Sha1)
		}
		if len(branch.CommitGitIDs) == 0 {
			branch.CommitGitIDs = make(map[library.Sha1]library.Sha256)
		}
		branch.CommitGitIDs[data.GID] = ""
	}
	r.Branches[branch.Name] = branch
	//get full data for each commit

	//get blobs
	return
}

func getRepoName(rname, path string) (string, error) {
	var name string
	if len(rname) > 0 {
		name = rname
	} else {
		directoryName, err := getLastDirectoryName(path)
		if err != nil {
			return "", fmt.Errorf("a repo name was not provided and a name could not be produced from the supplied repo path")
		}
		name = directoryName
	}
	if err := validateRepoName(name); err != nil {
		actors.LogCLI("repo name is invalid, we will use a slugified version instead", 3)
		name = slugify(name)
	}
	return name, nil
}

func validateRepoName(input string) error {
	// Check if string has more than 100 ASCII code points
	if len(input) > 100 {
		return fmt.Errorf("string exceeds the maximum length of 100 ASCII code points")
	}
	// Check if string contains any invalid characters
	valid := regexp.MustCompile(`^[-_.a-zA-Z0-9]+$`).MatchString(input)
	if !valid {
		return fmt.Errorf("string contains invalid characters")
	}
	return nil
}

func slugify(input string) string {
	// Remove leading and trailing whitespaces
	input = strings.TrimSpace(input)

	// Convert string to lowercase
	input = strings.ToLower(input)

	// Replace non-alphanumeric characters with hyphen (-)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	input = re.ReplaceAllString(input, "-")

	// Remove consecutive hyphens
	input = strings.Trim(input, "-")

	return input
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	// If the file does not exist or there is an error, return false
	if os.IsNotExist(err) {
		return false
	}

	// If the file exists, return true
	return true
}

func pathIsDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("%s does not exist\n", path)
		} else {
			fmt.Printf("Error accessing %s: %v\n", path, err)
		}
		return false
	}

	// Check if the file mode is a directory
	if fileInfo.Mode().IsDir() {
		return true
	}

	return false
}

func getLastDirectoryName(dirPath string) (string, error) {
	// Get the last element of the split path
	name := filepath.Base(dirPath)

	// Check if name is a directory
	if !filepath.IsAbs(name) {
		return name, nil
	}

	return "", fmt.Errorf("failed to get the name of the last directory")
}

type NewRepoOptions struct {
	Path         string
	Name         string
	UpstreamDTag string
	ForkedAt     string
	Simulate     bool //use simulations and never publish anything
}
