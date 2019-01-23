package fileinfo

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	p := New()
	info, skipReason := p.GetInfo(makeArgs("dir1/main.go",
		`package main
		
		func main(){
		}`,
	))
	assert.Equal(t, "", skipReason)
	assert.Equal(t, "Go", info.Language)
}

func makeArgs(filePath string, content string) InfoArgs {
	args := InfoArgs{}
	args.FilePath = filePath
	args.Content = []byte(content)
	args.Lines = bytes.Split(args.Content, []byte("\n"))
	return args
}

var testOKFilePath = "main.go"
var testOKContent = `package main
	
	func main(){
	}`

func TestFilePaths(t *testing.T) {
	cases := []struct {
		Path       string
		SkipReason string
	}{
		{"dir1/a.go", ""},
		{"config.json", skipConfigFile},
		{".config", skipDotFile},
		{"go.sum", skipBlacklisted},
		{"dependencies/a.go", skipVendoredFile},
		// enry will incorrectly match something like:
		// src/com/foo/android/cache/DiskLruCache.java
		// as a vendored file but it's not
		// we hardcore fix using src in path
		{"src/com/foo/android/cache/DiskLruCache.java", ""},
	}
	p := New()
	for _, c := range cases {
		_, skipReason := p.GetInfo(makeArgs(c.Path, testOKContent))
		if skipReason != c.SkipReason {
			t.Errorf("wanted skip reason %v for path %v, got %v", c.SkipReason, c.Path, skipReason)
		}
	}
}

func BenchmarkFilePaths(b *testing.B) {
	p := New()
	for i := 0; i < b.N; i++ {
		p.GetInfo(makeArgs(strings.Repeat("dir1/", 20)+"a.go", testOKContent))
	}
}

func makeArgsWithContentLen(l int) InfoArgs {
	content := make([]byte, l)
	for i := 0; i < len(content); i++ {
		if i%1000 == 0 {
			content[i] = '\n'
		} else {
			content[i] = 'a'
		}
	}
	copy(content, testOKContent)
	return makeArgs(testOKFilePath, string(content))
}

func TestMaxFileSizeNotExceeded(t *testing.T) {
	p := New()
	_, skipReason := p.GetInfo(makeArgsWithContentLen(maxFileSize))

	assert.Equal(t, "", skipReason)
}

func TestMaxFileSizeExceeded(t *testing.T) {
	p := New()
	_, skipReason := p.GetInfo(makeArgsWithContentLen(maxFileSize + 1000))

	assert.Equal(t, "File size was 1001K which exceeds limit of 1000K", skipReason)
}
func TestMaxLinesExceeded(t *testing.T) {
	p := New()
	_, skipReason := p.GetInfo(makeArgs("a.go", strings.Repeat("\n", maxLinePerFile+100)))
	assert.Equal(t, "File has more than 40000 lines", skipReason)

}
func TestMaxLineWidthExceeded(t *testing.T) {
	p := New()
	_, skipReason := p.GetInfo(makeArgs("a.go", strings.Repeat("a", maxBytesPerLine+1)))
	assert.Equal(t, "File has a line width of 1097 which is greater than max of 1096", skipReason)
}

func TestLanguage1(t *testing.T) {
	p := New()
	info, skipReason := p.GetInfo(makeArgs("dir1/main.go",
		`package main
		
		func main(){
		}`,
	))
	assert.Equal(t, "", skipReason)
	assert.Equal(t, "Go", info.Language)
}

func TestLanguageUnknown(t *testing.T) {
	p := New()
	_, skipReason := p.GetInfo(makeArgs("a",
		``,
	))
	assert.Equal(t, skipLanguageUnknown, skipReason)
}
