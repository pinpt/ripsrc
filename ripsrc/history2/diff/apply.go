package diff

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Line struct {
	Line   []byte
	Commit string
}

// String returns compact string representation of line. Useful in tests to see output.
func (l Line) String() string {
	return l.Commit + ":" + string(l.Line)
}

type File struct {
	Lines []Line
}

// String returns compact string representation of file. Useful in tests to see output.
func (f File) String() string {
	out := []string{}
	for i, l := range f.Lines {
		out = append(out, strconv.Itoa(i)+":"+l.String())
	}
	return strings.Join(out, "\n")
}

func NewNilFile() File {
	return File{}
}

func Apply(file File, diff Diff, commit string) File {
	res := make([]Line, len(file.Lines))
	copy(res, file.Lines)

	remLine := func(i int) {
		res = append(res[:i], res[i+1:]...)
	}
	addLine := func(i int, data []byte) {
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
		return a.Contexts[0].Offset > b.Contexts[0].Offset
	})

	for _, h := range diff.Hunks {
		scanner := bufio.NewScanner(bytes.NewReader(h.Data))
		i := h.Contexts[0].Offset - 1
		if i == -1 {
			i = 0
		}

		for scanner.Scan() {
			b := scanner.Bytes()
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
			default:
				panic(fmt.Errorf("invalid patch prefix, line %s prefix %v", b, op))
			}
		}
		if err := scanner.Err(); err != nil {
			panic(err)
		}
	}

	return File{Lines: res}
}

func copyLines(lines []Line) (res []Line) {
	res = make([]Line, len(lines))
	copy(res, lines)
	return
}
