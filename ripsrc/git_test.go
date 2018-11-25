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
	assert.Equal("123_8a6189f6cae8b1afab3cf63ad611f9e5bb88422f.json.gz", c.getFilename("123", "/foo/bar.go"))
	assert.Equal("123_d500d241933c3446619730654c7acf3670227373.json.gz", c.getFilename("123", "foo bar.go"))
	assert.Equal("123_257352a2942388a2924bb6e7e353f5352181cb9d.json.gz", c.getFilename("123", fmt.Sprintf("foo%cbar.go", '\\')))
}
