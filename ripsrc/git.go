package ripsrc

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// CommitStatus is a commit status type
type CommitStatus string

const (
	// GitFileCommitStatusAdded is the added status
	GitFileCommitStatusAdded = CommitStatus("added")
	// GitFileCommitStatusModified is the modified status
	GitFileCommitStatusModified = CommitStatus("modified")
	// GitFileCommitStatusRemoved is the removed status
	GitFileCommitStatusRemoved = CommitStatus("removed")
)

func (s CommitStatus) String() string {
	return string(s)
}

// CommitFile is a specific detail around a file in a commit
type CommitFile struct {
	Filename    string
	Status      CommitStatus
	Renamed     bool
	RenamedFrom string
	RenamedTo   string
}

// Commit is a specific detail around a commit
type Commit struct {
	Dir     string
	SHA     string
	Email   string
	Files   map[string]*CommitFile
	Date    time.Time
	Ordinal int64
}

var (
	lend         = []byte("\n")
	commitPrefix = []byte("commit ")
	authorPrefix = []byte("Author: ")
	emailRegex   = regexp.MustCompile("<(.*)>")
	datePrefix   = []byte("Date: ")
	space        = []byte(" ")
	tab          = []byte("\t")
	rPrefix      = []byte("R")
)

func toCommitStatus(name []byte) CommitStatus {
	switch string(name) {
	case "A":
		{
			return GitFileCommitStatusAdded
		}
	case "D":
		{
			return GitFileCommitStatusRemoved
		}
	case "M", "R", "C", "MM":
		{
			return GitFileCommitStatusModified
		}
	}
	return GitFileCommitStatusModified
}

func parseDate(d string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, d)
	if err != nil {
		return time.Now(), fmt.Errorf("error parsing commit date `%v`. %v", d, err)
	}
	return t.UTC(), nil
}

func parseAuthorEmail(email string) string {
	m := emailRegex.FindString(email)
	// strip out the angle brackets
	return m[1 : len(m)-1]
}

// StreamCommits will stream all the commits to the returned channel and signal the done channel when completed
func StreamCommits(dir string, commits chan<- *Commit, done *sync.WaitGroup, errors chan<- error) error {
	var errout bytes.Buffer
	var cmd *exec.Cmd
	cmd = exec.Command("git", "log", "--raw", "--reverse", "--pretty=format:commit %H%nAuthor: %an <%ae>%nDate: %aI%nParent: %P%n%n", "--no-merges")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Dir = dir
	cmd.Stderr = &errout
	if err := cmd.Start(); err != nil {
		if strings.Contains(errout.String(), "does not have any commits yet") {
			return fmt.Errorf("no commits found in repo at %s", dir)
		}
		if strings.Contains(errout.String(), "Not a git repository") {
			return fmt.Errorf("not a valid git repo found in repo at %s", dir)
		}
		return fmt.Errorf("error running git log --all --raw --date=iso-strict in dir %s, %v", dir, err)
	}
	go func() {
		defer func() {
			out.Close()
			done.Done()
		}()
		var commit *Commit
		r := bufio.NewReader(out)
		ordinal := time.Now().Unix()
		for {
			buf, err := r.ReadBytes(lend[0])
			if err != nil {
				if err == io.EOF {
					break
				}
				errors <- err
				return
			}

			buf = buf[0 : len(buf)-1]
			if len(buf) == 0 {
				continue
			}
			if bytes.HasPrefix(buf, commitPrefix) {
				sha := string(buf[len(commitPrefix):])
				i := strings.Index(sha, " ")
				if i > 0 {
					// trim off stuff after the sha since we can get tag info there
					sha = sha[0:i]
				}
				// send the old commit and create a new one
				if commit != nil { // because we send when we detect the next commit
					commits <- commit
				}
				commit = &Commit{
					Dir:     dir,
					SHA:     string(sha),
					Files:   make(map[string]*CommitFile, 0),
					Ordinal: ordinal,
				}
				ordinal++
				continue
			}
			if bytes.HasPrefix(buf, datePrefix) {
				d := bytes.TrimSpace(buf[len(datePrefix):])
				t, err := parseDate(string(d))
				if err != nil {
					errors <- fmt.Errorf("error parsing commit %s in %s. %v", commit.SHA, dir, err)
					return
				}
				commit.Date = t.UTC()
				continue
			}
			if bytes.HasPrefix(buf, authorPrefix) {
				commit.Email = parseAuthorEmail(string(buf[len(authorPrefix):]))
				continue
			}
			if buf[0] == ':' {
				// :100644␠100644␠d1a02ae0...␠a452aaac...␠M␉·pandora/pom.xml
				tok1 := bytes.Split(buf, space)
				tok2 := bytes.Split(bytes.Join(tok1[4:], space), tab)
				action := tok2[0]
				paths := tok2[1:]
				if len(action) == 1 {
					fn := string(bytes.TrimLeft(paths[0], " "))
					commit.Files[fn] = &CommitFile{
						Filename: fn,
						Status:   toCommitStatus(action),
					}
				} else if bytes.HasPrefix(action, rPrefix) {
					fromFn := string(bytes.TrimLeft(paths[0], " "))
					toFn := string(bytes.TrimLeft(paths[1], " "))
					commit.Files[fromFn] = &CommitFile{
						Status:      GitFileCommitStatusRemoved,
						Filename:    fromFn,
						Renamed:     true,
						RenamedFrom: fromFn,
						RenamedTo:   toFn,
					}
					commit.Files[toFn] = &CommitFile{
						Status:      GitFileCommitStatusAdded,
						Filename:    toFn,
						Renamed:     true,
						RenamedFrom: fromFn,
						RenamedTo:   toFn,
					}
				} else {
					fn := string(bytes.TrimLeft(paths[0], " "))
					commit.Files[fn] = &CommitFile{
						Status:   toCommitStatus(action),
						Filename: fn,
					}
				}
			}
		}
		if commit != nil {
			commits <- commit
		}
	}()
	return nil
}
