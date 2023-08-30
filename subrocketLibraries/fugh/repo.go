package fugh

import (
	"fmt"
	"os/exec"
	"strings"
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
