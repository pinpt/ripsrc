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
	IsBinary bool
	Hunks    []Hunk
}

func (d Diff) PathOrPrev() string {
	if d.Path != "" {
		return d.Path
	}
	return d.PathPrev
}

// Hunk is a part of the diff describing change to a part of file.
type Hunk struct {
	Locations []HunkLocation
	Data      []byte
}

func (h Hunk) String() string {
	return string(h.Data)
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
	stParseDiff      = "stParseDiff"
	stParsingPreMeta = "stParsingPreMeta"
	stNextIsCurrName = "stNextIsCurrName"
	stNextIsContext  = "stNextIsContext"
	stInPatchLines   = "stInPatchLines"
)

type parser struct {
	content []byte
	state   string

	diff Diff

	preMeta         map[string]string
	currentContexts []HunkLocation
	currentLines    []byte

	res []Hunk

	wantedMeta []string
}

const metaRenameFrom = "rename from"
const metaRenameTo = "rename to"
const metaNewFile = "new file"
const metaDeletedFile = "deleted file"
const metaBinaryFiles = "Binary files"

func newParser(content []byte) *parser {
	p := &parser{}
	p.content = content
	p.wantedMeta = []string{metaRenameFrom, metaRenameTo, metaNewFile, metaDeletedFile, metaBinaryFiles}
	return p
}

func (p *parser) Parse() (res Diff) {
	if len(p.content) == 0 {
		return
	}

	p.state = stParseDiff
	p.preMeta = map[string]string{}

	scanner := bufio.NewScanner(bytes.NewReader(p.content))
	scanner.Buffer(nil, maxLine)
	for scanner.Scan() {
		p.line(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	p.finishHunk()

	p.diff.Hunks = p.res

	if p.preMeta[metaNewFile] != "" {
		p.diff.PathPrev = ""
	}

	if p.preMeta[metaDeletedFile] != "" {
		p.diff.Path = ""
		p.diff.Hunks = nil
	}

	if p.preMeta[metaRenameFrom] != "" {
		p.diff.PathPrev = p.preMeta[metaRenameFrom]
		if p.preMeta[metaRenameTo] == "" {
			panic("has rename from, but not rename to")
		}
		p.diff.Path = p.preMeta[metaRenameTo]
	}

	if p.preMeta[metaBinaryFiles] != "" {
		p.diff.IsBinary = true
	}

	res = p.diff

	return
}

func (p *parser) line(b []byte) {
	switch p.state {
	case stParseDiff:
		p.state = stParsingPreMeta

		var err error
		p.diff.PathPrev, p.diff.Path, err = parseDiffDecl(b)
		if err != nil {
			if err == errParseDiffDeclMerge {
				// will get name from diff later
				return
			} else if err == errParseDiffDeclRenameWithSpaces {
				// will get name from diff later
				return
			}
			panic(err)
		}
	case stParsingPreMeta:
		if !startsWith(b, "---") {
			for _, s := range p.wantedMeta {
				if startsWith(b, s+" ") {
					p.preMeta[s] = string(b[len(s)+1:])
				}
			}
		} else {
			p.state = stNextIsCurrName

		}
	case stNextIsCurrName:
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
	if len(p.currentContexts) != 0 {
		h := Hunk{}
		h.Locations = p.currentContexts
		h.Data = p.currentLines

		p.res = append(p.res, h)

	}

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
