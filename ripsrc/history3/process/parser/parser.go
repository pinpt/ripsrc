package parser

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

type Parser struct {
	r io.Reader

	res    chan Commit
	commit Commit
	change Change

	state state
	diff  []byte

	msgEmptyLines int
}

type state string

const stNotStarted = "stNotStarted"
const stParentsNext = "stParentsNext"
const stSkipAuthor = "stSkipAuthor"
const stSkipMessage = "stSkipMessage"
const stSkippingMessageDiffOrCommitNext = "stSkippingMessageDiffOrCommitNext"
const stInDiff = "stInDiff"
const stCommitNext = "stCommitNext"

type Commit struct {
	Hash          string
	Parents       []string
	MergeDiffFrom string
	Changes       []Change
}

func (c Commit) String() string {
	res := []string{}
	res = append(res, c.Hash)
	res = append(res, strings.Join(c.Parents, ","))
	for _, c := range c.Changes {
		res = append(res, c.String())
	}
	return strings.Join(res, "*\n")
}

type Change struct {
	Diff []byte
}

func (c Change) String() string {
	return string(c.Diff)
}

func New(r io.Reader) *Parser {
	p := &Parser{}
	p.r = r
	return p
}

const mb = 1000 * 1000
const maxLine = 100 * mb

func (s *Parser) Run(res chan Commit) error {
	defer close(res)

	s.res = res
	s.state = stNotStarted

	scanner := bufio.NewScanner(s.r)
	scanner.Buffer(nil, maxLine)
	for scanner.Scan() {
		line := scanner.Bytes()
		s.line(line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// handle empty log output
	if s.state != stNotStarted {
		s.endCommit()
	}
	return nil
}

func (s *Parser) line(b []byte) {
	switch s.state {
	case stNotStarted:
		s.parseCommitLine(b)
	case stParentsNext:
		if startsWith(b, mergePrefix) {
		} else {
			s.state = stSkipAuthor
			s.line(b)
		}
	case stSkipAuthor:
		s.state = stSkippingMessageDiffOrCommitNext
	case stSkippingMessageDiffOrCommitNext:
		if len(b) == 0 {
			// commit message
			return
		}
		if b[0] == ' ' {
			// commit message
			return
		}
		if s.isDiffStart(b) {
			s.startDiff(b)
		} else if startsWith(b, "commit ") {
			s.endCommit()
			s.state = stCommitNext
			s.line(b)
		} else {
			panic(fmt.Errorf("stSkippingMessageDiffOrCommitNext unexpected line, got %s %s", hex.EncodeToString(b), b))
		}
	case stInDiff:
		if len(b) == 0 {
			s.endCommit()
		} else if s.isDiffStart(b) {
			s.endDiff()
			s.startDiff(b)
		} else {
			s.diff = append(s.diff, b...)
			s.diff = append(s.diff, '\n')
		}
	case stCommitNext:
		s.parseCommitLine(b)
	default:
		panic(fmt.Errorf("unknown state %v", s.state))
	}
}

func (s *Parser) startDiff(b []byte) {
	s.state = stInDiff
	s.diff = []byte{}
	s.diff = append(s.diff, b...)
	s.diff = append(s.diff, '\n')
}

func (s *Parser) endDiff() {
	c := Change{}
	c.Diff = s.diff
	s.commit.Changes = append(s.commit.Changes, c)
}

func (s *Parser) endCommit() {
	if s.state == stInDiff {
		s.endDiff()
	}
	s.state = stCommitNext
	s.res <- s.commit
}

func (s *Parser) isDiffStart(b []byte) bool {
	return len(b) > 4 && string(b[0:4]) == "diff"
}

func (s *Parser) parseCommitLine(b []byte) {
	prefix := "commit "
	if !startsWith(b, prefix) {
		panic(fmt.Errorf("no '%v' prefix, line %s", prefix, b))
	}
	data := string(b[len(prefix):])
	c := Commit{}
	if !strings.Contains(data, " ") {
		c.Hash = data
	} else {
		parts := strings.SplitN(data, " ", 2)
		if len(parts) != 2 {
			panic(fmt.Errorf("invalid format for commit line %s got len parts %v", b, len(parts)))
		}
		c.Hash = parts[0]
		fromPrefix := "(from "
		c.MergeDiffFrom = parts[1][len(fromPrefix) : len(parts[1])-1]
	}
	s.commit = c
	s.state = stParentsNext
}

const mergePrefix = "Merge: "

func startsWith(b []byte, prefix string) bool {
	if len(prefix) > len(b) {
		return false
	}
	return string(b[:len(prefix)]) == prefix
}

func (s *Parser) RunGetAll() (_ []Commit, err error) {
	res := make(chan Commit)
	done := make(chan bool)
	go func() {
		err = s.Run(res)
		done <- true
	}()
	var res2 []Commit
	for c := range res {
		res2 = append(res2, c)
	}
	<-done
	return res2, err
}
