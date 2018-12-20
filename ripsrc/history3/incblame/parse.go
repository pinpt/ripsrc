package incblame

import (
	"bufio"
	"bytes"
	"fmt"
)

// Diff is change made to one file.
type Diff struct {
	PathPrev string
	Path     string
	Hunks    []Hunk
}

// Hunk is a part of the diff describing change to a part of file.
type Hunk struct {
	Locations []HunkLocation
	Data      []byte
}

// HunkLocation is the operation, offset and line modified.
type HunkLocation struct {
	Op     OpType
	Offset int
	Lines  int
}

// OpType is type of change performed by hunk.
type OpType rune

const (
	// OpAdd is adding piece of code
	OpAdd OpType = '+'
	// OpDel is deleting piece of code
	OpDel = '-'
)

// Parse parses patch output for one file extracted from the following command.
// git log -p -c (sames as git-diff-tree -p -c)
// Returns parsed diff which could be applied to blame data saved in File.
func Parse(content []byte) (res Diff) {
	p := newParser(content)
	return p.Parse()
}

const (
	stLookingForPrevName = "stLookingForPrevName"
	stNextIsCurrName     = "stNextIsCurrName"
	stNextIsContext      = "stNextIsContext"
	stInPatchLines       = "stInPatchLines"
)

type parser struct {
	content []byte
	state   string

	diff Diff

	currentContexts []HunkLocation
	currentLines    []byte

	res []Hunk

	//endNl bool // does content ends with newline?
}

func newParser(content []byte) *parser {
	p := &parser{}
	p.content = content
	return p
}

func (p *parser) Parse() (res Diff) {
	if len(p.content) == 0 {
		return
	}
	//	if p.content[len(p.content)-1] == '\n' {
	//		p.endNl = true
	//	}

	p.state = stLookingForPrevName

	scanner := bufio.NewScanner(bytes.NewReader(p.content))
	for scanner.Scan() {
		p.line(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	p.finishHunk()
	res = p.diff
	res.Hunks = p.res
	return
}

func (p *parser) line(b []byte) {
	switch p.state {
	case stLookingForPrevName:
		if startsWith(b, "---") {
			p.diff.PathPrev = extractName(b)
			p.state = stNextIsCurrName
		}
	case stNextIsCurrName:
		p.diff.Path = extractName(b)
		p.state = stNextIsContext
	case stNextIsContext:
		if len(b) <= 2 {
			return
		}
		if string(b[0:2]) == "@@" {
			p.parseContext(b)
			return
		}
	case stInPatchLines:
		p.lineInPatchLines(b)
	default:
		panic("invalid state")
	}
}

func extractName(b []byte) string {
	if len(b) <= 4 {
		panic(fmt.Errorf("expected path name, line %s", string(b)))
	}
	name := string(b[4:])
	if name == "/dev/null" {
		return ""
	}
	if len(name) <= 2 {
		panic(fmt.Errorf("expected path name, line %s", string(b)))
	}
	return name[2:]
}

func (p *parser) finishHunk() {
	h := Hunk{}
	h.Locations = p.currentContexts
	h.Data = p.currentLines
	p.res = append(p.res, h)

	p.currentContexts = nil
	p.currentLines = nil
}

func (p *parser) parseContext(b []byte) {
	p.currentContexts = parseContext(b)
	p.state = stInPatchLines
}

func (p *parser) lineInPatchLines(b []byte) {
	if string(b[0:2]) == "@@" {
		p.finishHunk()
		p.parseContext(b)
		return
	}
	p.currentLines = append(p.currentLines, b...)
	p.currentLines = append(p.currentLines, "\n"...)
}
