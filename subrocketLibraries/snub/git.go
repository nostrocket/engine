package snub

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"nostrocket/engine/actors"
	"nostrocket/engine/library"
)

func GetFirstTreeSHA(repoPath string) (string, error) {
	return getFirstTreeSHA(repoPath)
}

func getFirstTreeSHA(repoPath string) (string, error) {
	cmd := exec.Command("git", "cat-file", "-p", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute git cat-file command: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "tree") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1], nil
			}
		}
	}
	return "", fmt.Errorf("failed to find tree identifier in git cat-file output")
}

func GetBlobIdentifiers(repoPath, treeIdentifier string) (blobIdentifiers []string, err error) {
	return getBlobIdentifiers(repoPath, treeIdentifier)
}

// getBlobIdentifiers returns a list of blob identifiers present in the given tree
func getBlobIdentifiers(repoPath, treeIdentifier string) (blobIdentifiers []string, err error) {
	cmd := exec.Command("git", "-C", repoPath, "cat-file", "-p", treeIdentifier)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "blob") {
			fields := strings.Fields(line)
			if len(fields[2]) == 40 && fields[1] == "blob" {
				blobIdentifiers = append(blobIdentifiers, fields[2])
			}
		}
	}

	return
}

func GetBinaryBlob(repoPath string, blobID string) ([]byte, error) {
	cmd := exec.Command("git", "-C", repoPath, "cat-file", "-p", blobID)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return output, nil
}

func CreateBlobMap(repoPath string) (map[string][]byte, error) {
	treeID, err := getFirstTreeSHA(repoPath)
	if err != nil {
		return nil, err
	}

	blobMap := make(map[string][]byte)
	err = iterateTree(repoPath, treeID, blobMap)
	if err != nil {
		return nil, err
	}

	return blobMap, nil
}

func iterateTree(repoPath string, treeID string, blobMap map[string][]byte) error {
	blobIDs, err := getBlobIdentifiers(repoPath, treeID)
	if err != nil {
		return err
	}

	for _, blobID := range blobIDs {
		blob, err := GetBinaryBlob(repoPath, blobID)
		if err != nil {
			return err
		}

		blobMap[blobID] = blob
	}

	cmd := exec.Command("git", "-C", repoPath, "cat-file", "-p", treeID)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fmt.Println(107)
		fmt.Println(line)
		if strings.Contains(line, "tree") {
			fields := strings.Fields(line)
			if len(fields[2]) == 40 && fields[1] == "tree" {
				err := iterateTree(repoPath, fields[2], blobMap)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func GetCurrentHeadCommitID(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	commitID := strings.TrimSpace(string(output))
	return commitID, nil
}

func GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "symbolic-ref", "--short", "HEAD")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Failed to get current branch: %v", err)
	}

	branch := strings.TrimSpace(out.String())
	return branch, nil
}

func getAllCommitIDs(r *git.Repository, commitID string) ([]string, error) {
	// Resolve the commit object based on the commit ID
	commitHash := plumbing.NewHash(commitID)
	commit, err := r.CommitObject(commitHash)
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve commit object: %v", err)
	}

	// Set to store unique commit IDs
	commitIDs := make(map[string]struct{})
	commitIDs[commitID] = struct{}{}

	// Define a recursive function to iterate over all parents
	var iterateParents func(commit *object.Commit)
	iterateParents = func(commit *object.Commit) {
		// Add the current commit ID to the set
		commitIDs[commit.Hash.String()] = struct{}{}

		// Iterate over the parents
		for _, parent := range commit.ParentHashes {
			parentCommit, err := r.CommitObject(parent)
			if err != nil {
				actors.LogCLI(err, 3)
				continue
			}

			// Recursively call the function for each parent commit
			//todo remove 10k limit after testing complete
			if len(commitIDs) < 10000 {
				iterateParents(parentCommit)
			}
		}
	}

	// Start iterating over parents from the given commit
	iterateParents(commit)

	// Convert the set to a slice
	var result []string
	for commitID := range commitIDs {
		result = append(result, commitID)
	}

	return result, nil
}

func getCommitData(commitSha1 string) (*Commit, error) {
	commitCmd := exec.Command("git", "cat-file", "-p", commitSha1)
	output, err := commitCmd.Output()
	if err != nil {
		return nil, err
	}

	commitData := parseCommitOutput(string(output))
	commitData.GID = commitSha1
	return commitData, nil
}

func parseCommitOutput(output string) *Commit {
	lines := strings.Split(output, "\n")
	commit := &Commit{}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "author") {
			authorLine := strings.TrimPrefix(line, "author ")
			authorData := strings.Split(authorLine, " <")
			commit.Author.Name = authorData[0]
			commit.Author.Email = strings.TrimSuffix(authorData[1], ">")
		} else if strings.HasPrefix(line, "committer") {
			committerLine := strings.TrimPrefix(line, "committer ")
			committerData := strings.Split(committerLine, " <")
			commit.Committer.Name = committerData[0]
			commit.Committer.Email = strings.TrimSuffix(committerData[1], ">")
		} else if strings.HasPrefix(line, "message") {
			commit.Message = strings.TrimPrefix(line, "message ")
		} else if strings.HasPrefix(line, "parent") {
			parentID := strings.TrimPrefix(line, "parent ")
			commit.ParentIDs = append(commit.ParentIDs, library.Sha1(parentID))
		} else if strings.HasPrefix(line, "tree") {
			commit.TreeID = library.Sha1(strings.TrimPrefix(line, "tree "))
		}
	}
	//
	//commit := &Commit{}
	//commit.GID = library.Sha1(lines[0][5:])
	//
	//authorLine := strings.Split(lines[1], " ")
	//fmt.Printf("\n%#v\n", authorLine)
	//commit.Name = LegacyIdentification{
	//	Name:    authorLine[1],
	//	Email:     authorLine[2][1 : len(authorLine[2])-1],
	//	Timestamp: parseTimestamp(authorLine[3]),
	//	UTCoffset: authorLine[4],
	//}
	//
	//committerLine := strings.Split(lines[2], " ")
	//commit.Committer = LegacyIdentification{
	//	Name:    committerLine[1],
	//	Email:     committerLine[2][1 : len(committerLine[2])-1],
	//	Timestamp: parseTimestamp(committerLine[3]),
	//	UTCoffset: committerLine[4],
	//}
	//
	//commit.Message = strings.Join(lines[4:], "\n")
	//
	//parentIDs := strings.Split(lines[3][8:], " ")
	//commit.ParentIDs = make([]library.Sha1, len(parentIDs))
	//for i, parentID := range parentIDs {
	//	commit.ParentIDs[i] = library.Sha1(parentID)
	//}
	//
	//commit.TreeID = library.Sha1(strings.Split(lines[5][5:], " ")[0])

	return commit
}

func parseTimestamp(timestampStr string) int64 {
	parseInt, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return 0
	}
	return parseInt
}

func getCommitData2(commitSha1 string, repo *git.Repository) (*Commit, error) {
	commitObj, err := repo.CommitObject(plumbing.NewHash(commitSha1))
	if err != nil {
		return nil, err
	}

	commitData := &Commit{}
	commitData.GID = commitObj.Hash.String()
	commitData.Author = parseSignature(commitObj.Author)
	commitData.Committer = parseSignature(commitObj.Committer)
	commitData.Message = commitObj.Message
	//commitData.ParentIDs = commitObj.ParentHashes
	for _, hash := range commitObj.ParentHashes {
		commitData.ParentIDs = append(commitData.ParentIDs, hash.String())
	}
	commitData.TreeID = commitObj.TreeHash.String()
	//commitData.LegacyBackup = commitObj.File()
	if len(commitObj.PGPSignature) > 0 {
		lines := strings.Split(commitObj.PGPSignature, "\n")
		var formattedSig string
		var linesAfterPGP int64
		for i, line := range lines {
			if linesAfterPGP > 0 {
				line = " " + line
				linesAfterPGP++
			}
			if linesAfterPGP == 0 {
				if strings.Contains(line, "-----BEGIN PGP SIGNATURE-----") {
					linesAfterPGP = 1
				}
			}
			formattedSig += line
			if i != len(lines)-1 {
				formattedSig += "\n"
			}
		}
		commitData.LegacySig = formattedSig
	}
	return commitData, nil
}

func parseSignature(sig object.Signature) LegacyIdentification {
	legacySig := LegacyIdentification{
		Name:      sig.Name,
		Email:     sig.Email,
		Timestamp: sig.When.Unix(),
		UTCoffset: sig.When.Format("-0700"),
	}
	return legacySig
}

func openRepository(path string) (*git.Repository, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("could not open local repository at %s", path)
	}
	return repo, nil
}

func getGitHashForObject(input string, t string) (string, error) {
	cmd := exec.Command("git", "hash-object", "-t"+t, "--stdin")
	//cmd.Stdin = ioutil.NopCloser(os.Stdin)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println(349)
		return "", err
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(56)
		return "", err
	}

	if err := cmd.Start(); err != nil {
		fmt.Println(61)
		return "", err
	}

	if _, err := stdin.Write([]byte(input)); err != nil {
		fmt.Println(66)
		return "", err
	}
	stdin.Close()

	output, err := ioutil.ReadAll(stdout)
	if err != nil {
		fmt.Println(73)
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		fmt.Println(cmd.String())
		return "", err
	}

	return string(output[:len(output)-1]), nil
}
