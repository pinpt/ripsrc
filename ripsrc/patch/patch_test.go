package patch

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	assert := assert.New(t)
	for _, test := range []struct {
		patch     string
		text      string
		expected  string
		checkline int
	}{
		{"@@ -0,1 +1,2 @@\n a\n+b\n", "a\n", "a\nb\n", 1},
		{"@@ -1,2 +1,3 @@\n a\n b\n+c\n", "a\nb\n", "a\nb\nc\n", 2},
		{"@@ -0,0 +1,2 @@\n+a\n+b\n", "", "a\nb\n", 0},
		{"@@ -0,0 +1,2 @@\n+a\n b\n", "", "a\n", 0},
		{"@@ -1,1 +1,1 @@\n x\n+a\n", "x\n", "x\na\n", 1},
		{"@@ -1,2 +1,1 @@\n x\n+a\n", "x\n", "x\na\n", 1},
		{"@@ -1,1 +1,1 @@\n \n-x\n", "\nx\n", "\n", -1},
		{"@@ -1,1 +1,1 @@\n a\n b\n+c\n", "a\nb\n", "a\nb\nc\n", 2},
		{"@@ -1,4 +1,3 @@\n a\n b\n-c\n d\n", "a\nb\nc\nd\n", "a\nb\nd\n", -1},
		{"@@ -1,4 +1,2 @@\n a\n b\n-c\n-d\n", "a\nb\nc\nd\n", "a\nb\n", -1},
		{"@@ -1,4 +1,1 @@\n-a\n b\n-c\n-d\n", "a\nb\nc\nd\n", "b\n", -1},
		{"@@ -1,4 +1,1 @@\n-a\n-b\n+x\n-c\n-d\n", "a\nb\nc\nd\n", "x\n", 0},
		{"@@ -1,4 +1,3 @@\n a\n-b\n+x\n-c\n d\n", "a\nb\nc\nd\n", "a\nx\nd\n", 1},
		{"@@ -1,4 +1,5 @@\n a\n-b\n+x\n-c\n d\n+e\n+f\n", "a\nb\nc\nd\n", "a\nx\nd\ne\nf\n", 1},
		{"@@ -0,0 +1,1 @@\n+a\n\\ No newline at end of file\n", "", "a", 0},
		{"@@ -3,7 +3,7 @@\n a\n b\n c\n-d\n+e\n f\n g\n-h\n+h\n\\ No newline at end of file\n", "x\nx\na\nb\nc\nd\nf\ng\nh", "x\nx\na\nb\nc\ne\nf\ng\nh", 5},
		{"@@ -3,2 +3,4 @@\n 3\n 4\n-5\n+a\n+b\n 6\n@@ -8,2 +8,3 @@\n 8\n+c\n", "1\n2\n3\n4\n5\n6\n7\n8\n9\n", "1\n2\n3\n4\na\nb\n6\n7\n8\nc\n9\n", 4},
	} {
		p := New("test")
		assert.NoError(p.Parse(test.patch))
		assert.Equal(test.patch, p.String())
		f := NewFile("test")
		assert.NoError(f.Parse(test.text, "oldcommit"))
		nf := p.Apply(f, "newcommit")
		assert.Equal(test.expected, nf.String())
		if test.checkline != -1 {
			val, ok := nf.Lines[test.checkline].Commit.(string)
			assert.True(ok)
			assert.Equal("newcommit", val)
		} else if len(nf.Lines) > 0 {
			val, ok := nf.Lines[0].Commit.(string)
			assert.True(ok)
			assert.Equal("oldcommit", val)
		}
	}
}

func TestParseRules(t *testing.T) {
	assert := assert.New(t)
	p := New("a.diff")
	patchfile, err := ioutil.ReadFile("testdata/a.diff")
	assert.NoError(err)
	assert.NoError(p.Parse(string(patchfile)))
	f := NewFile("a-in.txt")
	afile, err := ioutil.ReadFile("testdata/a-in.txt")
	assert.NoError(err)
	assert.NoError(f.Parse(string(afile), nil))
	// fmt.Println(f.Stringify(true))
	nf := p.Apply(f, nil)
	bfile, err := ioutil.ReadFile("testdata/a-out.txt")
	assert.NoError(err)
	assert.Equal(string(bfile), nf.String())
}
