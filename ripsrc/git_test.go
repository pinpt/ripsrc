package ripsrc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToCommitStatus(t *testing.T) {
	assert := assert.New(t)
	tt := []struct {
		data   []byte
		answer string
	}{
		{[]byte("A"), "added"},
		{[]byte("D"), "removed"},
		{[]byte("M"), "modified"},
		{[]byte("MM"), "modified"},
		{[]byte("T"), "modified"},
	}
	for _, v := range tt {
		response := toCommitStatus(v.data)
		assert.Contains(response, v.answer)
	}
}

func TestParseDate(t *testing.T) {
	assert := assert.New(t)
	tt := []struct {
		data   string
		answer string
	}{
		{"2018-09-26", ""},
	}
	for _, v := range tt {
		_, err := parseDate(v.data)
		assert.Error(err)
	}
}

func TestParseEmail(t *testing.T) {
	assert := assert.New(t)
	tt := []struct {
		data   string
		answer string
	}{
		{"^cf89534 <jhaynie@pinpt.com> 2018-06-30 17:06:10 -0700   1", "jhaynie@pinpt.com"},
		{"<someone@somewhere.com>", "someone@somewhere.com"},
		{"<[someone@somewhere.com]>", "someone@somewhere.com"},
		{"\\", ""},
		{"", ""},
	}
	for _, v := range tt {
		response := parseEmail(v.data)
		assert.Equal(response, v.answer)
	}
}

func TestGetFilename(t *testing.T) {
	assert := assert.New(t)
	tt := []struct {
		data   string
		answer string
	}{
		{"/somewhere/somewhere/somewhere/something.txt => /somewhere/somewhere/something.txt", "/somewhere/somewhere/something.txt"},
		{`file.go\{file.go => newfile.go\}newfile.go`, `file.go\/newfile.go\/newfile.go`},
		{"", ""},
		{"/", "/"},
	}
	for _, v := range tt {
		response, _, _ := getFilename(v.data)
		assert.Equal(response, v.answer)
	}
}

func TestGetFilenameEscaping(t *testing.T) {
	assert := assert.New(t)
	var c commitFileHistory
	assert.Equal("123__foo_bar_go.json.gz", c.getFilename("123", "/foo/bar.go"))
	assert.Equal("123_foo_bar_go.json.gz", c.getFilename("123", "foo bar.go"))
	assert.Equal("123_foo_bar_go.json.gz", c.getFilename("123", fmt.Sprintf("foo%cbar.go", '\\')))
}
