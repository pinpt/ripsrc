package commitmeta

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pinpt/ripsrc/ripsrc/gitexec"
)

type Opts struct {
	CommitFromIncl string

	// AllBranches set to true to process all branches. If false, processes commits reachable from HEAD only.
	AllBranches bool
}

type Processor struct {
	repoDir    string
	gitCommand string
	opts       Opts
}

func New(repoDir string, opts Opts) *Processor {
	s := &Processor{
		repoDir:    repoDir,
		gitCommand: "git",
		opts:       opts,
	}
	return s
}

// Commit is a specific detail around a commit
type Commit struct {
	//Dir            string
	SHA            string
	AuthorName     string
	AuthorEmail    string
	CommitterName  string
	CommitterEmail string
	Files          map[string]*CommitFile
	Date           time.Time
	Ordinal        int64
	Message        string

	Parents []string
	Signed  bool
	//Previous *Commit
}

// Author returns either the author name (preference) or the email if not found
func (c Commit) Author() string {
	if c.AuthorName != "" {
		return c.AuthorName
	}
	return c.AuthorEmail
}

// CommitFile is a specific detail around a file in a commit
type CommitFile struct {
	Filename    string
	Status      CommitStatus
	Renamed     bool
	Copied      bool
	RenamedFrom string
	RenamedTo   string
	CopiedFrom  string
	Additions   int
	Deletions   int
	Binary      bool
}

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

func (s *Processor) RunSlice() (res []Commit, _ error) {
	resChan := make(chan Commit)
	done := make(chan bool)
	go func() {
		for r := range resChan {
			res = append(res, r)
		}
		done <- true
	}()
	err := s.Run(resChan)
	<-done
	return res, err
}

func (s *Processor) RunMap() (map[string]Commit, error) {
	res := map[string]Commit{}
	resChan := make(chan Commit)
	done := make(chan bool)
	go func() {
		for r := range resChan {
			res[r.SHA] = r
		}
		done <- true
	}()
	err := s.Run(resChan)
	<-done
	return res, err
}

func (s *Processor) Run(res chan Commit) error {
	defer close(res)
	r, err := s.gitLog()
	if err != nil {
		return err
	}
	defer r.Close()

	var parser parser
	parser.dir = s.repoDir
	//parser.limit = limit
	parser.commits = res

	// we don't need this in new code. TODO: check and remove
	fjChan := make(chan *CommitFile, 100)
	go func() {
		for range fjChan {

		}
	}()
	parser.filejobs = fjChan

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		ok, err := parser.parse(scanner.Text())
		if err != nil {
			return fmt.Errorf("error processing commit from %v. %v", s.repoDir, err)
		}
		if !ok {
			break
		}
	}
	if parser.commit != nil && parser.commit.SHA != "" { // because we send when we detect the next commit
		res <- *parser.commit
	}

	return nil
}

func (s *Processor) gitLog() (io.ReadCloser, error) {
	// empty file at tem location to set an empty attributesFile
	f, err := ioutil.TempFile("", "ripsrc")
	if err != nil {
		return nil, err
	}
	err = f.Close()
	if err != nil {
		return nil, err
	}

	args := []string{
		"-c", "core.attributesFile=" + f.Name(),
		"-c", "diff.renameLimit=10000",
		"log",
		"-c",
		"--raw",
		"--reverse",
		"--numstat",
		"--pretty=format:!SHA: %H%n!Parents: %P%n!Committer: %ce%n!CName: %cn%n!Author: %ae%n!AName: %an%n!Signed-Key: %GK%n!Date: %aI%n!Message: %s%n",
	}

	if s.opts.AllBranches {
		args = append(args, "--all")
	}

	if s.opts.CommitFromIncl != "" {
		args = append(args, s.opts.CommitFromIncl+"^..HEAD")
	}

	return gitexec.ExecPiped(context.Background(), s.gitCommand, s.repoDir, args)
}

var (
	commitPrefix        = []byte("!SHA: ")
	authorPrefix        = []byte("!Author: ")
	authorNamePrefix    = []byte("!AName: ")
	committerPrefix     = []byte("!Committer: ")
	committerNamePrefix = []byte("!CName: ")
	signedEmailPrefix   = []byte("!Signed-Key: ")
	messagePrefix       = []byte("!Message: ")
	parentsPrefix       = []byte("!Parents: ")
	emailRegex          = regexp.MustCompile("<(.*)>")
	emailBracketsRegex  = regexp.MustCompile("^\\[(.*)\\]$")
	datePrefix          = []byte("!Date: ")
	space               = []byte(" ")
	tab                 = []byte("\t")
	removePrefix        = []byte("R")
	copyPrefix          = []byte("C")
	filenameMask        = regexp.MustCompile("^(100644|100755)$")
	deletedMask         = []byte("000000")
	renameRe            = regexp.MustCompile("(.*)\\{(.*) => (.*)\\}(.*)")
)

func toCommitStatus(name []byte) CommitStatus {
	switch string(name) {
	case "A":
		return GitFileCommitStatusAdded
	case "D":
		return GitFileCommitStatusRemoved
	case "M", "R", "C", "MM", "T":
		return GitFileCommitStatusModified
	}
	panic("unknown commit status: " + string(name))
}

func parseDate(d string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, d)
	if err != nil {
		return time.Now(), fmt.Errorf("error parsing commit date `%v`. %v", d, err)
	}
	return t, nil
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

	// allow our test case to change the executable
	gitCommand, _ = exec.LookPath("git")
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

type parserState int

const (
	parserStateHeader parserState = iota
	parserStateFiles
	parserStateNumStats
	parserStateDiff
)

type parser struct {
	commits  chan<- Commit
	filejobs chan<- *CommitFile
	commit   *Commit
	dir      string
	limit    int
	total    int
	ordinal  int64
	state    parserState
}

func (p *parser) parse(line string) (bool, error) {
	if line == "" {
		return true, nil
	}
	buf := []byte(line)
	for {
		switch p.state {
		case parserStateHeader:
			if bytes.HasPrefix(buf, commitPrefix) {
				sha := string(buf[len(commitPrefix):])
				i := strings.Index(sha, " ")
				if i > 0 {
					// trim off stuff after the sha since we can get tag info there
					sha = sha[0:i]
				}
				//var parent *string
				//if p.commit != nil {
				//	parent = &p.commit.SHA
				//}
				//var parentCommit *Commit
				// send the old commit and create a new one
				if p.commit != nil && p.commit.SHA != "" { // because we send when we detect the next commit
					//parentCommit = p.commit
					p.commits <- *p.commit
					p.commit = nil
				}
				if p.limit > 0 && p.total == p.limit {
					p.commit = nil
					return false, nil
				}
				p.ordinal++
				p.commit = &Commit{
					//Dir:      p.dir,
					SHA:     string(sha),
					Files:   make(map[string]*CommitFile, 0),
					Ordinal: p.ordinal,
					//Parent:   parent,
					//Previous: parentCommit,
				}
				p.total++
				return true, nil
			}
			if bytes.HasPrefix(buf, parentsPrefix) {
				parents := string(buf[len(parentsPrefix):])
				if len(parents) != 0 {
					p.commit.Parents = strings.Split(parents, " ")
				}
			}
			if bytes.HasPrefix(buf, datePrefix) {
				d := bytes.TrimSpace(buf[len(datePrefix):])
				t, err := parseDate(string(d))
				if err != nil {
					return false, fmt.Errorf("error parsing commit %s in %s. %v", p.commit.SHA, p.dir, err)
				}
				p.commit.Date = t
				return true, nil
			}
			if bytes.HasPrefix(buf, authorPrefix) {
				p.commit.AuthorEmail = string(buf[len(authorPrefix):])
				return true, nil
			}
			if bytes.HasPrefix(buf, authorNamePrefix) {
				p.commit.AuthorName = string(buf[len(authorNamePrefix):])
				return true, nil
			}
			if bytes.HasPrefix(buf, committerPrefix) {
				p.commit.CommitterEmail = string(buf[len(committerPrefix):])
				return true, nil
			}
			if bytes.HasPrefix(buf, committerNamePrefix) {
				p.commit.CommitterName = string(buf[len(committerNamePrefix):])
				return true, nil
			}
			if bytes.HasPrefix(buf, signedEmailPrefix) {
				signedCommitLine := string(buf[len(signedEmailPrefix):])
				if signedCommitLine != "" {
					p.commit.Signed = true
				}
				return true, nil
			}
			if bytes.HasPrefix(buf, messagePrefix) {
				p.commit.Message = string(buf[len(messagePrefix):])
				p.state = parserStateFiles
				return true, nil
			}
		case parserStateFiles:
			if buf[0] == ':' {
				// :100644␠100644␠d1a02ae0...␠a452aaac...␠M␉·pandora/pom.xml
				tok1 := bytes.Split(buf, space)
				mask := tok1[1]
				// fmt.Println(p.commit.SHA, line, string(mask))
				// if the mask isn't a regular file or deleted file, skip it
				if !filenameMask.Match(mask) && !bytes.Equal(mask, deletedMask) {
					return true, nil
				}
				tok2 := bytes.Split(bytes.Join(tok1[4:], space), tab)
				action := tok2[0]
				paths := tok2[1:]
				if len(action) == 1 {
					fn := string(bytes.TrimLeft(paths[0], " "))
					cf := &CommitFile{
						Filename: fn,
						Status:   toCommitStatus(action),
					}
					p.commit.Files[fn] = cf
					p.filejobs <- cf
				} else if bytes.HasPrefix(action, removePrefix) {
					// rename
					fromFn := string(bytes.TrimLeft(paths[0], " "))
					toFn := string(bytes.TrimLeft(paths[1], " "))
					// panic("renaming [" + fromFn + "] => [" + toFn + "]")
					p.commit.Files[fromFn] = &CommitFile{
						Status:      GitFileCommitStatusRemoved,
						Filename:    fromFn,
						Renamed:     true,
						RenamedFrom: fromFn,
						RenamedTo:   toFn,
					}
					cf := &CommitFile{
						Status:      GitFileCommitStatusAdded,
						Filename:    toFn,
						Renamed:     true,
						RenamedFrom: fromFn,
						RenamedTo:   toFn,
					}
					p.commit.Files[toFn] = cf
					p.filejobs <- cf
				} else if bytes.HasPrefix(action, copyPrefix) {
					// copy a file into a new file ... it's basically a new file
					fromFn := string(bytes.TrimLeft(paths[0], " "))
					toFn := string(bytes.TrimLeft(paths[1], " "))
					cf := &CommitFile{
						Status:     GitFileCommitStatusAdded,
						Filename:   toFn,
						Copied:     true,
						CopiedFrom: fromFn,
					}
					p.commit.Files[toFn] = cf
					p.filejobs <- cf
				} else {
					fn := string(bytes.TrimLeft(paths[0], " "))
					cf := &CommitFile{
						Status:   toCommitStatus(action),
						Filename: fn,
					}
					p.commit.Files[fn] = cf
					p.filejobs <- cf
				}
				return true, nil
			}
			p.state = parserStateNumStats
			continue
		case parserStateNumStats:
			tok := bytes.Split(buf, tab)
			// handle the file stats output
			if len(tok) == 3 {
				tok := regSplit(string(buf), tabSplitter)
				fn, _, _ := getFilename(tok[2])
				file := p.commit.Files[fn]
				// fmt.Println("num stats", fn, "=>", oldfn, "renamed", renamed, "is nil?", file == nil)
				if file == nil {
					// BUG: this needs fixing, next 4 lines is a quick workaround
					// Special case for merge
					// TODO: parse lines like this properly:
					// ::100644 100644 100644 1dbddb0... 4cd4b38... 904d55b... MM      main.go
					file = &CommitFile{}
					file.Filename = fn
					file.Status = GitFileCommitStatusModified
					p.commit.Files[fn] = file

					// this is OK, just means it was a special entry such as directory only, skip this one
					// return true, nil
				}
				if tok[0] == "-" {
					file.Binary = true
				} else {
					adds, _ := strconv.ParseInt(tok[0], 10, 32)
					dels, _ := strconv.ParseInt(tok[1], 10, 32)
					file.Additions = int(adds)
					file.Deletions = int(dels)
				}
			} else {
				p.state = parserStateDiff
				continue
			}
		case parserStateDiff:
			if line[0:1] == "!" {
				p.state = parserStateHeader
				continue
			}
		}
		break
	}
	return true, nil
}
