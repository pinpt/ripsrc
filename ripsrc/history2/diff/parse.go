package diff

import (
	"bufio"
	"bytes"
)

type OpType rune

const (
	OpAdd OpType = '+'
	OpDel        = '-'
)

type HunkContext struct {
	Op     OpType
	Offset int
	Lines  int
}

type Hunk struct {
	Contexts []HunkContext
	Data     []byte
}

type Diff struct {
	Hunks []Hunk
}

func Parse(content []byte) (res Diff) {
	p := newParser(content)
	return p.Parse()
}

const (
	stLookingForContext = "looking-for-context"
	stInPatchLines      = "in-patch-lines"
)

type parser struct {
	content []byte
	state   string

	currentContexts []HunkContext
	currentLines    []byte

	res []Hunk

	//endNl bool // does content ends with newline?
}

func newParser(content []byte) *parser {
	p := &parser{}
	p.content = content
	p.state = stLookingForContext
	return p
}

func (p *parser) Parse() (res Diff) {
	if len(p.content) == 0 {
		return
	}
	//	if p.content[len(p.content)-1] == '\n' {
	//		p.endNl = true
	//	}

	scanner := bufio.NewScanner(bytes.NewReader(p.content))
	for scanner.Scan() {
		p.line(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	p.finishHunk()
	res.Hunks = p.res
	return
}

func (p *parser) line(b []byte) {
	switch p.state {
	case stLookingForContext:
		p.lineLookingForContext(b)
	case stInPatchLines:
		p.lineInPatchLines(b)
	default:
		panic("invalid state")
	}
}

func (p *parser) finishHunk() {
	h := Hunk{}
	h.Contexts = p.currentContexts
	h.Data = p.currentLines
	p.res = append(p.res, h)

	p.currentContexts = nil
	p.currentLines = nil
}

func (p *parser) lineLookingForContext(b []byte) {
	if len(b) <= 2 {
		return
	}
	if string(b[0:2]) == "@@" {
		p.parseContext(b)
		return
	}
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
