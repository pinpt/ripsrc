package ripsrc

import (
	"fmt"
	"strings"

	"github.com/pinpt/ripsrc/ripsrc/patch"
)

type parsestate int

const (
	lookingForHeader parsestate = iota
	lookingForStart
	insidePatch
)

// diff has one or more patch files contained in the diff set
type diff struct {
	files    []*patch.Patch
	state    parsestate
	patchbuf strings.Builder
	cur      *patch.Patch
}

// newDiffParser returns a new diff parser
func newDiffParser() *diff {
	return &diff{
		files: make([]*patch.Patch, 0),
		state: lookingForHeader,
	}
}

const (
	diffheaderPrefix    = "diff --git "
	contextHeaderPrefix = "@@ "
)

func (d *diff) String() string {
	lines := make([]string, 0)
	for _, f := range d.files {
		lines = append(lines, f.String())
	}
	return strings.Join(lines, "\n")
}

func (d *diff) reset() {
	d.files = make([]*patch.Patch, 0)
	d.state = lookingForHeader
	d.patchbuf.Reset()
	d.cur = nil
}

func (d *diff) complete() error {
	if d.cur != nil {
		if err := d.cur.Parse(d.patchbuf.String()); err != nil {
			return fmt.Errorf("error parsing patch: %v", err)
		}
		d.cur = nil
		d.patchbuf.Reset()
	}
	return nil
}

func (d *diff) parse(line string) (bool, error) {
	var ok bool
	for {
		switch d.state {
		case lookingForHeader:
			if strings.HasPrefix(line, diffheaderPrefix) {
				d.state = lookingForStart
				i := strings.Index(line, " b/")
				if i < 0 {
					return false, fmt.Errorf("expected filename but couldn't find it: %v", line)
				}
				if d.cur != nil {
					if err := d.cur.Parse(d.patchbuf.String()); err != nil {
						return false, fmt.Errorf("error parsing patch: %v", err)
					}
				}
				d.cur = patch.New(strings.TrimSpace(line[i+3:]))
				d.files = append(d.files, d.cur)
				d.patchbuf.Reset()
				ok = true
				break
			}
			break
		case lookingForStart:
			if strings.HasPrefix(line, contextHeaderPrefix) {
				d.state = insidePatch
				d.patchbuf.WriteString(line)
				d.patchbuf.WriteString("\n")
			}
			if strings.HasPrefix(line, diffheaderPrefix) {
				d.state = lookingForHeader
				continue
			}
		case insidePatch:
			if strings.HasPrefix(line, diffheaderPrefix) {
				d.state = lookingForHeader
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
