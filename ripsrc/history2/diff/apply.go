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
	Commit string
	Lines  []Line
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

func ApplyMerge(parents []File, diff Diff, commit string) File {
	//fmt.Println("parent1")
	//fmt.Println(parents[0])
	//fmt.Println("parent2")
	//fmt.Println(parents[1])
	res := make([]Line, len(parents[0].Lines))
	copy(res, parents[0].Lines)

	remLine := func(i int) {
		res = append(res[:i], res[i+1:]...)
	}
	addLine := func(i int, data []byte, commit string) {
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

		lenp := len(parents)
		// offsets are used to retrieve corresponding line owner from merge parent
		offsets := make([]int, lenp)
		for i := 0; i < lenp; i++ {
			offsets[i] = h.Contexts[i].Offset - 2
		}

		for scanner.Scan() {
			b := scanner.Bytes()
			if len(b) == 0 {
				panic(fmt.Errorf("could not process patch line, it was empty, h.Data %v", string(h.Data)))
			}
			if len(b) < len(parents) {
				panic(fmt.Errorf("could not process patch line, len < len(parentsy, h.Data %v", string(h.Data)))
			}

			ops := b[0:lenp]
			data := b[lenp:]

			for i, v := range ops[1:] {
				i += 1
				switch v {
				case ' ', '\t':
					// if was removed from another parent, and was not in this one at all
					removedFromAnother := false
					for _, v := range ops {
						if v == '-' {
							removedFromAnother = true
						}
					}
					if removedFromAnother {
						// this line was not in parent, if it would be in parent it would be marked as - as well
					} else {
						offsets[i]++
					}
				case '-':
					offsets[i]--
				case '+':
				default:
					panic(fmt.Errorf("offsets invalid patch prefix, line '%s' prefix '%v' '%s'", b, v, string(v)))
				}
			}

			//fmt.Println("offsets", offsets, "line", string(b))

			switch ops[0] {
			case ' ', '\t':
				// no change
				i++
			case '-':
				remLine(i)
				// no need to inc offset
			case '+':
				srcI := 0
				for i, op := range ops {
					if op == ' ' {
						srcI = i
					}
				}
				src := ""
				if srcI == 0 {
					// source is merge itself
					src = commit
				} else {
					offset := offsets[srcI]
					lines := parents[srcI].Lines
					if offset < 0 || offset >= len(lines) {
						panic(fmt.Errorf("invalid offset for merge parent, wanted %v, len(lines)=%v", offset, len(lines)))
					}
					src = lines[offset].Commit
				}
				addLine(i, data, src)
				i++
			default:
				panic(fmt.Errorf("invalid patch prefix, line %s prefix '%v' '%s'", b, ops[0], string(ops[0])))
			}
		}
		if err := scanner.Err(); err != nil {
			panic(err)
		}
	}

	return File{Lines: res, Commit: commit}
}

func applySingleParent(file File, diff Diff, commit string) File {
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
			if len(b) == 0 {
				panic(fmt.Errorf("could not process patch line, it was empty, h.Data %v", string(h.Data)))
			}
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

	return File{Lines: res, Commit: commit}
}

func copyLines(lines []Line) (res []Line) {
	res = make([]Line, len(lines))
	copy(res, lines)
	return
}
