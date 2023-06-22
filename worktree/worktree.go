package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ridge/must/v2"
)

var cachedDirty bool
var dirtyOnce sync.Once

var cachedCurrentVersion string
var currentVersionOnce sync.Once

// CurrentCommitID returns the current commit id
func CurrentCommitID() string {
	// Jenkins
	if id, ok := os.LookupEnv("GIT_COMMIT"); ok {
		return id
	}

	idCmd := exec.Command("git", "rev-parse", "HEAD")
	idCmd.Stderr = os.Stderr

	return strings.Trim(string(must.OK1(idCmd.Output())), "\n")
}

func currentCommitTimestamp() time.Time {
	timestampCmd := exec.Command("git", "show", "--format=%ct", "--no-patch")
	timestampCmd.Stderr = os.Stderr

	stringTimestamp := strings.Trim(string(must.OK1(timestampCmd.Output())), "\n")

	return time.Unix(must.OK1(strconv.ParseInt(stringTimestamp, 10, 64)), 0)
}

// This function calculates a hash of the whole checked out worktree, including
// modified files.
//
// This is done by creating a separate git index file, adding the contents of
// the whole worktree to it and then writing it to the repository. ID of this
// object is the hash of the worktree.
//
// This causes some garbage accumulating in the repository, but it is cleaned up
// by a periodic 'git gc', and Git does not have an in-memory "simulate writing
// to repository" mode to avoid it.
func worktreeID() string {
	tempDir := must.OK1(os.MkdirTemp("", "worktree-id-"))
	defer func() {
		_ = os.RemoveAll(tempDir) // we don't care about failed cleanup
	}()
	tempIndexFile := tempDir + "/index"

	gitAddCmd := exec.Command("git", "add", "--all")
	gitAddCmd.Stdout = os.Stdout
	gitAddCmd.Stderr = os.Stderr
	gitAddCmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+tempIndexFile)
	must.OK(gitAddCmd.Run())

	gitWriteTreeCmd := exec.Command("git", "write-tree")
	gitWriteTreeCmd.Stderr = os.Stderr
	gitWriteTreeCmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+tempIndexFile)
	return strings.Trim(string(must.OK1(gitWriteTreeCmd.Output())), "\n")
}

func semver(commitID string, commitTimestamp time.Time, dirtyWorktreeID string) string {
	ver := fmt.Sprintf("0.0.0-%s-%s", commitTimestamp.UTC().Format("20060102150405"), commitID[:12])
	if dirtyWorktreeID != "" {
		ver += "-dirty-" + dirtyWorktreeID
	}
	return ver
}

// Dirty returns whether the checked out worktree is dirty (contains new or modified files)
func Dirty() bool {
	dirtyOnce.Do(func() {
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusCmd.Stderr = os.Stderr
		cachedDirty = len(must.OK1(statusCmd.Output())) != 0
	})
	return cachedDirty
}

// CurrentVersion returns the current version of the worktree in SemVer 2.0
// format. Dirty trees get a stable version too (two identically-dirty trees
// will produce an identical version). Debug versions are marked as such as
// well.
func CurrentVersion() string {
	currentVersionOnce.Do(func() {
		commitID := CurrentCommitID()
		commitTimestamp := currentCommitTimestamp()
		dirtyWorktreeID := ""
		if Dirty() {
			dirtyWorktreeID = worktreeID()
		}
		cachedCurrentVersion = semver(commitID, commitTimestamp, dirtyWorktreeID)
	})
	return cachedCurrentVersion
}

func main() {
	fmt.Println(CurrentVersion())
}
