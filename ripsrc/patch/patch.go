package patch

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type operationType int

const (
	operationAdd operationType = iota
	operationDel
	operationEq
)

func (t operationType) String() string {
	switch t {
	case operationAdd:
		return "+"
	case operationDel:
		return "-"
	}
	return " "
}

type operation struct {
	action operationType
	line   string
	offset int
}

func (o operation) String() string {
	return fmt.Sprintf("operation[action=%s,offset=%d,line=%v", o.action, o.offset, o.line)
}

type context struct {
	start1  int
	length1 int
	start2  int
	length2 int
}

type hunk struct {
	context    context
	operations []operation
	noeol      bool
}

// Line is a specific line in a file
type Line struct {
	Buffer string
	Commit interface{} // allow this to be set by the caller
}

// File is an abstraction over a set of lines in a file
type File struct {
	Name  string
	Lines []*Line
	noeol bool
}

func (f *File) String() string {
	return f.Stringify(false)
}

// Stringify will allow you to stringify the file with or without line numbers
func (f *File) Stringify(linenums bool) string {
	if f == nil {
		panic("passed an invalid File object")
	}
	lines := make([]string, 0)
	if f.Lines != nil {
		var lf string
		if linenums {
			lf = fmt.Sprintf(`%%0%dd|%%s`, len(fmt.Sprintf("%d", len(f.Lines))))
		}
		for i, line := range f.Lines {
			if linenums {
				lines = append(lines, fmt.Sprintf(lf, 1+i, line.Buffer))
			} else {
				lines = append(lines, line.Buffer)
			}
		}
	}
	out := strings.Join(lines, "\n")
	if f.noeol && out[len(out)-1:] == "\n" {
		out = out[0 : len(out)-1]
	}
	return out
}

type commit interface {
	CommitSHA() string
	Author() string
	CommitDate() time.Time
}

func padRight(str string, length int, pad byte) string {
	l := len(str)
	if l >= length {
		max := length - 1
		return str[0:max] + fmt.Sprintf("%c", pad)
	}
	buf := bytes.NewBufferString(str)
	for i := 0; i < length-len(str); i++ {
		buf.WriteByte(pad)
	}
	return buf.String()
}

// Blame will print a blame style output
func (f *File) Blame() string {
	lines := make([]string, 0)
	if f.Lines != nil {
		for i, line := range f.Lines {
			if commit, ok := line.Commit.(commit); ok {
				sha := commit.CommitSHA()
				ts := commit.CommitDate()
				author := commit.Author()
				lines = append(lines, fmt.Sprintf("%s (%s %s %d) %s", sha[0:6], padRight(author, 15, ' '), ts.UTC().Format(time.RFC822Z), 1+i, line.Buffer))
			}
		}
	}
	out := strings.Join(lines, "\n")
	if f.noeol && out[len(out)-1:] == "\n" {
		out = out[0 : len(out)-1]
	}
	return out
}

// NewFile returns a new file object
func NewFile(name string) *File {
	return &File{name, make([]*Line, 0), false}
}

// Parse will parse the buffer into a set of lines inside the file
func (f *File) Parse(buf string, commit interface{}) error {
	lines := strings.Split(buf, "\n")
	for _, line := range lines {
		f.Lines = append(f.Lines, &Line{line, commit})
	}
	return nil
}

// Patch describes a set of changes
type Patch struct {
	Filename string
	hunks    []*hunk
}

func (p *Patch) String() string {
	var buf strings.Builder
	for _, hunk := range p.hunks {
		buf.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", hunk.context.start1, hunk.context.length1, hunk.context.start2, hunk.context.length2))
		for _, o := range hunk.operations {
			switch o.action {
			case operationAdd:
				buf.WriteString("+")
			case operationDel:
				buf.WriteString("-")
			case operationEq:
				buf.WriteString(" ")
			}
			buf.WriteString(o.line)
			buf.WriteString("\n")
		}
		if hunk.noeol {
			buf.WriteString("\\ No newline at end of file\n")
		}
	}
	return buf.String()
}

// Apply will apply a patch to an existing file for a new commit and return the new File with merged patch
func (p *Patch) Apply(file *File, commit interface{}) *File {
	newfile := NewFile(p.Filename)
	if file == nil {
		newfile.Lines = make([]*Line, 0)
	} else {
		newfile.Lines = make([]*Line, len(file.Lines))
		// make a new object so as not to mutate the original
		for i, l := range file.Lines {
			newfile.Lines[i] = &Line{l.Buffer, l.Commit}
		}
	}
	for _, hunk := range p.hunks {
		for _, o := range hunk.operations {
			offset := o.offset
			if Debug {
				fmt.Println(o)
			}
			switch o.action {
			case operationAdd:
				if offset > len(newfile.Lines) {
					if Debug {
						fmt.Println("adding new line at", offset, ">>>", o.line, "len", len(newfile.Lines))
					}
					newfile.Lines = append(newfile.Lines, &Line{o.line, commit})
				} else {
					if Debug {
						fmt.Println("adding line at", offset, ">>>", o.line, "len", len(newfile.Lines))
					}
					newfile.Lines = append(newfile.Lines[:offset], append([]*Line{&Line{o.line, commit}}, newfile.Lines[offset:]...)...)
				}
			case operationDel:
				if offset+1 > len(newfile.Lines) {
					fmt.Println(p.Filename, o)
					fmt.Println(file)
					fmt.Println("--> current lines=>", newfile.Stringify(true))
					fmt.Println(hunk.context)
					panic(fmt.Sprintf("invalid patch for %v. need to delete line %d but only has %d lines", p.Filename, offset, len(newfile.Lines)))
				}
				if Debug {
					fmt.Println("removing line at", offset, ">>>", newfile.Lines[offset], "len", len(newfile.Lines))
				}
				newfile.Lines = append(newfile.Lines[:offset], newfile.Lines[offset+1:]...)
			}
			offset++
		}
		if hunk.noeol {
			newfile.noeol = true
		}
	}
	return newfile
}

// Parse will parse the patch text into this current path object
func (p *Patch) Parse(buf string) error {
	toks := strings.Split(buf, "\n")
	state := parseStateStart
	var start, offset, delta, count int
	var currentHunk *hunk
	for _, tok := range toks {
		for {
			switch state {
			case parseStateStart:
				if contextRE.MatchString(tok) {
					state = parseStateBody
					currentHunk = &hunk{
						context: parsePatchHeader(tok),
					}
					count++
					start = currentHunk.context.start1
					if start > 0 {
						start--
					}
					if Debug {
						fmt.Println("start", start, "offset", offset, "delta", delta, "ctx", currentHunk.context)
					}
					offset = delta
					p.hunks = append(p.hunks, currentHunk)
				}
			case parseStateBody:
				if len(tok) > 0 {
					op := tok[0:1]
					if Debug {
						fmt.Println("op", op, "start", start, "offset", offset, "delta", delta, "line", tok[1:], currentHunk.context.length1)
					}
					switch op {
					case "+":
						currentHunk.operations = append(currentHunk.operations, operation{operationAdd, tok[1:], start + offset})
						offset++
						delta++
					case "-":
						currentHunk.operations = append(currentHunk.operations, operation{operationDel, tok[1:], start + offset})
						delta--
					case " ":
						currentHunk.operations = append(currentHunk.operations, operation{operationEq, tok[1:], start + offset})
						offset++
					case "\\":
						currentHunk.noeol = true
						offset++
					default:
						if contextRE.MatchString(tok) {
							state = parseStateStart
							continue
						}
						offset++
					}
				}
			}
			break
		}
	}
	return nil
}

// New returns a new empty Patch
func New(filename string) *Patch {
	return &Patch{Filename: filename}
}

var contextRE = regexp.MustCompile("^@@ [-](\\d+),?(\\d+)?\\s[+](\\d+),?(\\d+)? @@")

func parsePatchHeader(line string) context {
	tok := contextRE.FindAllStringSubmatch(line, -1)
	start1, length1, start2, length2 := tok[0][1], tok[0][2], tok[0][3], tok[0][4]
	var s1, l1, s2, l2 int
	if length1 == "" {
		val, _ := strconv.Atoi(start1)
		start1 = fmt.Sprintf("%d", val-1)
		length1 = "1"
	}
	if length2 == "" {
		val, _ := strconv.Atoi(start2)
		start2 = fmt.Sprintf("%d", val-1)
		length2 = "1"
	}
	s1, _ = strconv.Atoi(start1)
	l1, _ = strconv.Atoi(length1)
	s2, _ = strconv.Atoi(start2)
	l2, _ = strconv.Atoi(length2)
	return context{s1, l1, s2, l2}
}

type parseState int

const (
	parseStateStart parseState = iota
	parseStateBody
)

// Debug turns on verbose debug output
var Debug = false
