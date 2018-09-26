package ripsrc

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"strconv"
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
	Additions   int
	Deletions   int
	Binary      bool
}

// Callback for handling the commit job
type Callback func(err error, result *BlameResult, total int)

// Commit is a specific detail around a commit
type Commit struct {
	Dir            string
	SHA            string
	AuthorEmail    string
	CommitterEmail string
	Files          map[string]*CommitFile
	Date           time.Time
	Ordinal        int64
	Message        string
	Parent         *string
	Signed         bool

	callback Callback
}

var (
	commitPrefix       = []byte("commit ")
	authorPrefix       = []byte("Author: ")
	committerPrefix    = []byte("Committer: ")
	signedEmailPrefix  = []byte("Signed-Email: ")
	messagePrefix      = []byte("Message: ")
	parentPrefix       = []byte("Parent: ")
	emailRegex         = regexp.MustCompile("<(.*)>")
	emailBracketsRegex = regexp.MustCompile("^\\[(.*)\\]$")
	datePrefix         = []byte("Date: ")
	space              = []byte(" ")
	tab                = []byte("\t")
	rPrefix            = []byte("R")
	filenameMask       = []byte("100644")
	deletedMask        = []byte("000000")
	renameRe           = regexp.MustCompile("(.*)\\{(.*) => (.*)\\}(.*)")
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

func parseEmail(email string) string {
	// strip out the angle brackets
	if emailRegex.MatchString(email) {
		m := emailRegex.FindStringSubmatch(email)
		s := m[1]
		// attempt to strip out square brackets if found
		if emailBracketsRegex.MatchString(s) {
			m = emailBracketsRegex.FindStringSubmatch(s)
			return m[1]
		}
		return s
	}
	return ""
}

func getFilename(fn string) (string, string, bool) {
	if renameRe.MatchString(fn) {
		match := renameRe.FindStringSubmatch(fn)
		// use path.Join to remove empty directories and to correct join paths
		// must be path not filepath since it's always unix style in git and on windows
		// filepath will use \
		oldfn := path.Join(match[1], match[2], match[4])
		newfn := path.Join(match[1], match[3], match[4])
		return newfn, oldfn, true
	}
	// straight rename without parts
	s := strings.Split(fn, " => ")
	if len(s) > 1 {
		return s[1], s[0], true
	}
	return fn, fn, false
}

var (
	tabSplitter        = regexp.MustCompile("\\t")
	spaceSplitter      = regexp.MustCompile("[ ]")
	whitespaceSplitter = regexp.MustCompile("\\s+")
)

func regSplit(text string, splitter *regexp.Regexp) []string {
	indexes := splitter.FindAllStringIndex(text, -1)
	laststart := 0
	result := make([]string, len(indexes)+1)
	for i, element := range indexes {
		result[i] = text[laststart:element[0]]
		laststart = element[1]
	}
	result[len(indexes)] = text[laststart:len(text)]
	return result
}

// streamCommits will stream all the commits to the returned channel and block until completed
func streamCommits(ctx context.Context, dir string, sha string, limit int, commits chan<- Commit, errors chan<- error) error {
	errout := getBuffer()
	defer putBuffer(errout)
	var cmd *exec.Cmd
	args := []string{
		"log",
		"--raw",
		"--reverse",
		"--numstat",
		"--pretty=format:commit %H%nCommitter: %ce%nAuthor: %ae%nSigned-Email: %GS%nDate: %aI%nParent: %P%nMessage: %s%n",
		"--no-merges",
	}
	// if provided, we need to start streaming after this commit forward
	if sha != "" {
		args = append(args, sha+"...")
	}
	// fmt.Println(args)
	cmd = exec.CommandContext(ctx, "git", args...)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer out.Close()
	cmd.Dir = dir
	cmd.Stderr = errout
	if err := cmd.Start(); err != nil {
		if strings.Contains(errout.String(), "does not have any commits yet") {
			return fmt.Errorf("no commits found in repo at %s", dir)
		}
		if strings.Contains(errout.String(), "Not a git repository") {
			return fmt.Errorf("not a valid git repo found in repo at %s", dir)
		}
		return fmt.Errorf("error running git log in dir %s, %v", dir, err)
	}
	var total int
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var commit *Commit
		r := bufio.NewReaderSize(out, 200) // most lines are pretty small for the result based on test sampling of sizes
		ordinal := time.Now().Unix()
		s := bufio.NewScanner(r)
		scanbuf := getBuffer()
		defer putBuffer(scanbuf)
		s.Buffer(scanbuf.Bytes(), bufio.MaxScanTokenSize)
		for s.Scan() {
			if s.Err() != nil {
				if strings.Contains(s.Err().Error(), "file already closed") {
					break
				}
				errors <- fmt.Errorf("error reading while streaming commits from %v for sha %v. %v", dir, sha, s.Err())
				return
			}
			buf := s.Bytes()
			if len(buf) == 0 {
				continue
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
			// fmt.Println(string(buf))
			if bytes.HasPrefix(buf, commitPrefix) {
				sha := string(buf[len(commitPrefix):])
				i := strings.Index(sha, " ")
				if i > 0 {
					// trim off stuff after the sha since we can get tag info there
					sha = sha[0:i]
				}
				// send the old commit and create a new one
				if commit != nil { // because we send when we detect the next commit
					commits <- *commit
					commit = nil
				}
				if limit > 0 && total >= limit {
					commit = nil
					break
				}
				commit = &Commit{
					Dir:     dir,
					SHA:     string(sha),
					Files:   make(map[string]*CommitFile, 0),
					Ordinal: ordinal,
				}
				ordinal++
				total++
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
				commit.AuthorEmail = string(buf[len(authorPrefix):])
				continue
			}
			if bytes.HasPrefix(buf, committerPrefix) {
				commit.CommitterEmail = string(buf[len(committerPrefix):])
				continue
			}
			if bytes.HasPrefix(buf, signedEmailPrefix) {
				signedCommitLine := string(buf[len(signedEmailPrefix):])
				if signedCommitLine != "" {
					commit.Signed = true
					signedEmail := parseEmail(signedCommitLine)
					if signedEmail != "" {
						// if signed, mark it as such as use this as the preferred email
						commit.AuthorEmail = signedEmail
					}
				}
				continue
			}
			if bytes.HasPrefix(buf, messagePrefix) {
				commit.Message = string(buf[len(messagePrefix):])
				continue
			}
			if bytes.HasPrefix(buf, parentPrefix) {
				parent := string(buf[len(parentPrefix):])
				commit.Parent = &parent
				continue
			}
			if buf[0] == ':' {
				// fmt.Println(string(buf))
				// :100644␠100644␠d1a02ae0...␠a452aaac...␠M␉·pandora/pom.xml
				tok1 := bytes.Split(buf, space)
				mask := tok1[1]
				// if the mask isn't a regular file or deleted file, skip it
				if !bytes.Equal(mask, filenameMask) && !bytes.Equal(mask, deletedMask) {
					continue
				}
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
				continue
			}
			tok := bytes.Split(buf, tab)
			// handle the file stats output
			if len(tok) == 3 {
				tok := regSplit(string(buf), tabSplitter)
				fn, oldfn, renamed := getFilename(tok[2])
				file := commit.Files[fn]
				if file == nil {
					// this is OK, just means it was a special entry such as directory only, skip this one
					continue
				}
				if renamed {
					file.RenamedFrom = oldfn
					file.Renamed = true
				}
				if tok[0] == "-" {
					file.Binary = true
				} else {
					adds, _ := strconv.ParseInt(tok[0], 10, 32)
					dels, _ := strconv.ParseInt(tok[1], 10, 32)
					file.Additions = int(adds)
					file.Deletions = int(dels)
				}
			}
		}
		if commit != nil {
			select {
			case commits <- *commit:
				break
			default:
			}
		}
	}()
	if err := cmd.Wait(); err != nil {
		errors <- fmt.Errorf("error streaming commits from %v for sha %v. %v. %v", dir, sha, err, strings.TrimSpace(errout.String()))
		return nil
	}
	wg.Wait()
	return nil
}
