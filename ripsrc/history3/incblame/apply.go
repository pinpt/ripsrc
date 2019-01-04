package incblame

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Blame contains blame information for a file, with commit hash that created each particular line.
type Blame struct {
	Commit   string
	Lines    []Line
	IsBinary bool
}

func BlameBinaryFile(commit string) *Blame {
	return &Blame{Commit: commit, IsBinary: true}
}

// Line contains actual data and commit hash for each line in the file.
type Line struct {
	Line   []byte
	Commit string
}

// String returns compact string representation of line. Useful in tests to see output.
func (l Line) String() string {
	return l.Commit + ":" + string(l.Line)
}

func (l Line) Eq(l2 Line) bool {
	if l.Commit != l2.Commit {
		return false
	}
	if !bytes.Equal(l.Line, l2.Line) {
		return false
	}
	return true
}

// String returns compact string representation of file. Useful in tests to see output.
func (f Blame) String() string {
	out := []string{f.Commit}
	if len(f.Lines) == 0 {
		out = append(out, "empty")
	}
	if f.IsBinary {
		out = append(out, "binary")
	} else {
		for i, l := range f.Lines {
			out = append(out, strconv.Itoa(i)+":"+l.String())
		}
	}
	return strings.Join(out, "\n")
}

func (f Blame) Eq(f2 *Blame) bool {
	if f.Commit != f2.Commit {
		return false
	}
	if len(f.Lines) != len(f2.Lines) {
		return false
	}
	for i := range f.Lines {
		a := f.Lines[i]
		b := f2.Lines[i]
		if !a.Eq(b) {
			return false
		}
	}
	return true
}

func Apply(file Blame, diff Diff, commit string, fileForDebug string) Blame {
	rerr := func(err error) {
		panic(fmt.Errorf("commit:%v file:%v %v", commit, fileForDebug, err))
	}

	if diff.IsBinary {
		rerr(errors.New("can't apply diff which is binary file diff"))
	}

	res := make([]Line, len(file.Lines))
	copy(res, file.Lines)

	remLine := func(i int) {
		if i > len(res) {
			rerr(fmt.Errorf("trying to remove line which is not in blame, line %v blame %v", i, file))
		}
		res = append(res[:i], res[i+1:]...)
	}
	addLine := func(i int, data []byte) {
		if i > len(res) {
			rerr(fmt.Errorf("trying to add line at index %v, which is not in blame blame %v", i, file))
		}

		temp := []Line{}
		temp = append(temp, res[:i]...)
		temp = append(temp, Line{Line: data, Commit: commit})
		if i != len(res) {
			temp = append(temp, res[i:]...)
		}
		res = temp
	}

	sort.Slice(diff.Hunks, func(i, j int) bool {
		a := diff.Hunks[i]
		b := diff.Hunks[j]
		return a.Locations[0].Offset > b.Locations[0].Offset
	})

	for _, h := range diff.Hunks {
		if len(h.Locations) == 0 {
			rerr(fmt.Errorf("no location in diff hunk %+v", h))
		}

		scanner := bufio.NewScanner(bytes.NewReader(h.Data))
		scanner.Buffer(nil, maxLine)

		i := h.Locations[0].Offset - 1
		if i == -1 {
			i = 0
		}

		for scanner.Scan() {
			b := scanner.Bytes()
			if len(b) == 0 {
				rerr(fmt.Errorf("could not process patch line, it was empty, h.Data %v", string(h.Data)))
			}
			b = copyBytes(b)

			op := b[0]
			data := b[1:]
			switch op {
			case ' ', '\t':
				// no change
				i++
			case '-':
				remLine(i)
				// no need to inc offset
			case '+':
				addLine(i, data)
				i++
			case 92:
				if string(b) == "\\ No newline at end of file" {
					// can ignore this, we do not case about end of file newline
					continue
				}
				rerr(fmt.Errorf("invalid patch line, starts with \\ but not 'No newline at end of file', line '%s'", b))

			default:
				rerr(fmt.Errorf("invalid patch prefix, line %s prefix %v commit %v", b, op, commit))
			}
		}
		if err := scanner.Err(); err != nil {
			rerr(err)
		}
	}

	return Blame{Lines: res, Commit: commit}
}

func copyBytes(b []byte) []byte {
	res := make([]byte, len(b))
	copy(res, b)
	return res
}

func copyLines(lines []Line) (res []Line) {
	res = make([]Line, len(lines))
	copy(res, lines)
	return
}
