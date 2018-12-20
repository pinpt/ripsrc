package parser

import (
	"bufio"
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
}

type state string

const stNotStarted = "stNotStarted"
const stParentsNext = "stParentsNext"
const stDiffNext = "stDiffNext"
const stInDiff = "stInDiff"
const stCommitNext = "stCommitNext"

func New(r io.Reader) *Parser {
	p := &Parser{}
	p.r = r
	return p
}

func (s *Parser) Run(res chan Commit) error {
	defer close(res)

	s.res = res
	s.state = stNotStarted

	scanner := bufio.NewScanner(s.r)
	for scanner.Scan() {
		s.line(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	s.endCommit()

	return nil
}

func (s *Parser) line(b []byte) {
	switch s.state {
	case stNotStarted:
		s.parseCommit(b)
	case stParentsNext:
		s.parseParents(b)
	case stDiffNext:
		s.startDiff(b)
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
		s.parseCommit(b)
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
	s.endDiff()
	s.state = stCommitNext
	s.res <- s.commit
}

func (s *Parser) isDiffStart(b []byte) bool {
	return len(b) > 4 && string(b[0:4]) == "diff"
}

func (s *Parser) parseCommit(b []byte) {
	prefix := "!Hash: "
	if !startsWith(b, prefix) {
		panic(fmt.Errorf("no !Hash prefix, line %s", string(b)))
	}
	c := Commit{}
	c.Hash = string(b[len(prefix):])
	s.commit = c
	s.state = stParentsNext
}

func (s *Parser) parseParents(b []byte) {
	prefix := "!Parents: "
	if !startsWith(b, prefix) {
		panic(fmt.Errorf("no !Parents prefix, line %s", string(b)))
	}
	d := string(b[len(prefix):])
	if len(d) != 0 {
		s.commit.Parents = strings.Split(d, " ")
	}
	s.state = stDiffNext
}

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

type Commit struct {
	Hash    string
	Parents []string
	Changes []Change
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
