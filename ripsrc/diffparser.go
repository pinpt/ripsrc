package ripsrc

import (
	"fmt"
	"strings"

	"github.com/pinpt/ripsrc/ripsrc/patch"
)

type parsestate int

const (
	lookingForHeader parsestate = iota
	lookingForCommit
	lookingForStart
	insidePatch
)

type filehistory struct {
	SHA        string
	Patch      *patch.Patch
	CopyFrom   string
	RenameFrom string
	Deleted    bool
	Binary     bool
}

// diff has one or more patch files contained in the diff set
type diff struct {
	filename string
	state    parsestate
	patchbuf strings.Builder
	patch    *patch.Patch
	current  *filehistory
	history  []*filehistory
}

// newDiffParser returns a new diff parser
func newDiffParser(filename string) *diff {
	return &diff{
		filename: filename,
		history:  make([]*filehistory, 0),
		state:    lookingForCommit,
	}
}

const (
	diffheaderPrefix       = "diff --git "
	contextHeaderPrefix    = "@@ "
	copyFromHeaderPrefix   = "copy from "
	renameFromHeaderPrefix = "rename from "
	deletedHeaderPrefix    = "deleted file"
	binaryHeaderPrefix     = "Binary files "
)

func (d *diff) String() string {
	lines := make([]string, 0)
	for _, history := range d.history {
		lines = append(lines, fmt.Sprintf("%s%s\n", string(commitPrefix), history.SHA))
		lines = append(lines, history.Patch.String())
	}
	return strings.Join(lines, "\n")
}

func (d *diff) reset() {
	d.state = lookingForCommit
	d.history = make([]*filehistory, 0)
	d.current = nil
	d.patchbuf.Reset()
}

func (d *diff) complete() error {
	if err := d.patch.Parse(d.patchbuf.String()); err != nil {
		return fmt.Errorf("error parsing patch: %v", err)
	}
	return nil
}

func (d *diff) parse(line string) (bool, error) {
	var ok bool
	for {
		switch d.state {
		case lookingForCommit:
			if strings.HasPrefix(line, string(commitPrefix)) {
				commit := strings.TrimSpace(line[len(commitPrefix):])
				if d.patch != nil {
					if err := d.patch.Parse(d.patchbuf.String()); err != nil {
						return false, err
					}
					// fmt.Println("%%%%%%%%%%% ", d.filename, commit, d.patch)
					d.patchbuf.Reset()
				}
				d.state = lookingForHeader
				d.patch = patch.New(d.filename, commit)
				d.current = &filehistory{commit, d.patch, "", "", false, false}
				d.history = append(d.history, d.current)
			}
			break
		case lookingForHeader:
			if strings.HasPrefix(line, diffheaderPrefix) {
				d.state = lookingForStart
				break
			}
			break
		case lookingForStart:
			if strings.HasPrefix(line, copyFromHeaderPrefix) {
				d.current.CopyFrom = strings.TrimSpace(line[len(copyFromHeaderPrefix):])
			} else if strings.HasPrefix(line, renameFromHeaderPrefix) {
				d.current.RenameFrom = strings.TrimSpace(line[len(renameFromHeaderPrefix):])
			} else if strings.HasPrefix(line, contextHeaderPrefix) {
				d.state = insidePatch
				d.patchbuf.WriteString(line)
				d.patchbuf.WriteString("\n")
			} else if strings.HasPrefix(line, diffheaderPrefix) {
				d.state = lookingForHeader
				continue
			} else if strings.HasPrefix(line, deletedHeaderPrefix) {
				d.current.Deleted = true
			} else if strings.HasPrefix(line, binaryHeaderPrefix) {
				d.current.Binary = true
			}
		case insidePatch:
			if strings.HasPrefix(line, diffheaderPrefix) {
				d.state = lookingForHeader
				continue
			}
			if strings.HasPrefix(line, string(commitPrefix)) {
				d.state = lookingForCommit
				continue
			}
			d.patchbuf.WriteString(line)
			d.patchbuf.WriteString("\n")
		}
		ok = true
		break
	}
	return ok, nil
}
