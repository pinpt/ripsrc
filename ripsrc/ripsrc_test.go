package ripsrc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicSingleRip(t *testing.T) {
	assert := assert.New(t)
	results := make(chan BlameResult, 6)
	errors := make(chan error, 1)
	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "..")
	Rip(context.Background(), dir, results, errors, &Filter{SHA: "591377c17227ffa7134812188bbbda685e366b21"})
	// fmt.Println("after rip")
	select {
	case err := <-errors:
		assert.NoError(err)
	case result := <-results:
		// fmt.Println(result)
		assert.Equal("a0ec861c9e157908d06a814079abbb63e54784e3", result.Commit.SHA)
	default:
		assert.Fail("should have had result")
	}
}

func TestBasicMultiRip(t *testing.T) {
	assert := assert.New(t)
	results := make(chan BlameResult, 20)
	errors := make(chan error, 1)
	cwd, _ := os.Getwd()
	dir := filepath.Join(cwd, "..")
	var count int
	Rip(context.Background(), dir, results, errors, &Filter{SHA: "db285eb2a083c8d764a55841e6b83a0eb9130516"})
	var buf bytes.Buffer
Loop:
	for {
		select {
		case err := <-errors:
			assert.NoError(err)
		case result, ok := <-results:
			if ok {
				fmt.Fprintln(&buf, result.Commit.SHA, result.Filename, result.Language, result.Blanks, result.Comments, result.Loc, result.Size, result.Sloc, result.Complexity, result.WeightedComplexity, result.Status.String(), result.Commit.AuthorEmail, result.Commit.Date.String(), result.Commit.Signed, result.Commit.Message)
				count++
			} else {
				break Loop
			}
		default:
			break Loop
		}
	}
	assert.Equal(20, count)
	// fmt.Println(buf.String())
	assert.Equal(expendedMulti, buf.String())
}

var expendedMulti = `80d8ce43683663ed70d820c90ef3bfc0deee92c5 .gitignore  0 0 0 0 0 0 0 modified jhaynie@pinpt.com 2018-08-08 06:14:33 +0000 UTC true - be more strict when running blame: (a) check loc size (b) check bytes per line (c) total file size - move to using compiled version of github/gitignore for more robust file exclusions - move custom file exclusions to custom_patterns.txt to make it easier to maintain
80d8ce43683663ed70d820c90ef3bfc0deee92c5 Makefile Makefile 10 3 36 678 23 0 0 modified jhaynie@pinpt.com 2018-08-08 06:14:33 +0000 UTC true - be more strict when running blame: (a) check loc size (b) check bytes per line (c) total file size - move to using compiled version of github/gitignore for more robust file exclusions - move custom file exclusions to custom_patterns.txt to make it easier to maintain
80d8ce43683663ed70d820c90ef3bfc0deee92c5 custom_patterns.txt Text 20 0 195 2001 175 0 0 added jhaynie@pinpt.com 2018-08-08 06:14:33 +0000 UTC true - be more strict when running blame: (a) check loc size (b) check bytes per line (c) total file size - move to using compiled version of github/gitignore for more robust file exclusions - move custom file exclusions to custom_patterns.txt to make it easier to maintain
80d8ce43683663ed70d820c90ef3bfc0deee92c5 genignore.go Go 7 9 108 2635 92 26 0 added jhaynie@pinpt.com 2018-08-08 06:14:33 +0000 UTC true - be more strict when running blame: (a) check loc size (b) check bytes per line (c) total file size - move to using compiled version of github/gitignore for more robust file exclusions - move custom file exclusions to custom_patterns.txt to make it easier to maintain
80d8ce43683663ed70d820c90ef3bfc0deee92c5 ripsrc/blame.go Go 23 51 377 10709 303 47 0 modified jhaynie@pinpt.com 2018-08-08 06:14:33 +0000 UTC true - be more strict when running blame: (a) check loc size (b) check bytes per line (c) total file size - move to using compiled version of github/gitignore for more robust file exclusions - move custom file exclusions to custom_patterns.txt to make it easier to maintain
b4f5e5c5393ba6e02147566ec971f0e37c58c81e custom_patterns.txt Text 20 0 195 2000 175 0 0 modified jhaynie@pinpt.com 2018-08-08 14:16:31 +0000 UTC true fixed mistyped paste, more cleanup
b4f5e5c5393ba6e02147566ec971f0e37c58c81e ripsrc/blame.go Go 23 51 377 10720 303 48 0 modified jhaynie@pinpt.com 2018-08-08 14:16:31 +0000 UTC true fixed mistyped paste, more cleanup
92bb26225a5190f814359f918fa9d6d370a14b03 .gitignore  0 0 0 0 0 0 0 modified jhaynie@pinpt.com 2018-08-09 00:50:01 +0000 UTC true check in generated file so that we can just have as dependency without separate compile step
92bb26225a5190f814359f918fa9d6d370a14b03 ripsrc/gitignore.go  0 0 0 0 0 0 0 added jhaynie@pinpt.com 2018-08-09 00:50:01 +0000 UTC true check in generated file so that we can just have as dependency without separate compile step
90b2063ae9d6f4baf9dc0c3c07118242aae2f218 genignore.go Go 7 9 94 2210 78 22 0 modified jordaz@pinpt.com 2018-08-31 19:49:32 +0000 UTC false BE-885 using just custom patterns
1fdc27107a172c27be1804ad4365715750ab0d47 Gopkg.lock  0 0 0 0 0 0 0 modified jordaz@pinpt.com 2018-09-03 14:09:44 +0000 UTC false Upgrade scc from 1.4 to 1.9
1fdc27107a172c27be1804ad4365715750ab0d47 Gopkg.toml  0 0 0 0 0 0 0 modified jordaz@pinpt.com 2018-09-03 14:09:44 +0000 UTC false Upgrade scc from 1.4 to 1.9
1fdc27107a172c27be1804ad4365715750ab0d47 ripsrc/blame.go Go 23 51 378 10757 304 48 0 modified jordaz@pinpt.com 2018-09-03 14:09:44 +0000 UTC false Upgrade scc from 1.4 to 1.9
591377c17227ffa7134812188bbbda685e366b21 ripsrc/blame.go Go 24 51 383 11033 308 48 0 modified jordaz@pinpt.com 2018-09-04 16:13:22 +0000 UTC false Adding reasons why commit/commit_files are excluded
a0ec861c9e157908d06a814079abbb63e54784e3 Gopkg.lock  0 0 0 0 0 0 0 modified jhaynie@pinpt.com 2018-09-05 19:12:32 +0000 UTC true - fixed issue where the dot in the regexp was improperly being double escaped - improve the skip reason to be more specific - add a few test and benchmark cases - add go.sum for new go1.11 dependency file
a0ec861c9e157908d06a814079abbb63e54784e3 custom_patterns.txt Text 20 0 196 2020 176 0 0 modified jhaynie@pinpt.com 2018-09-05 19:12:32 +0000 UTC true - fixed issue where the dot in the regexp was improperly being double escaped - improve the skip reason to be more specific - add a few test and benchmark cases - add go.sum for new go1.11 dependency file
a0ec861c9e157908d06a814079abbb63e54784e3 genignore.go Go 7 9 94 2208 78 22 0 modified jhaynie@pinpt.com 2018-09-05 19:12:32 +0000 UTC true - fixed issue where the dot in the regexp was improperly being double escaped - improve the skip reason to be more specific - add a few test and benchmark cases - add go.sum for new go1.11 dependency file
a0ec861c9e157908d06a814079abbb63e54784e3 ripsrc/blame.go Go 24 51 396 11341 321 52 0 modified jhaynie@pinpt.com 2018-09-05 19:12:32 +0000 UTC true - fixed issue where the dot in the regexp was improperly being double escaped - improve the skip reason to be more specific - add a few test and benchmark cases - add go.sum for new go1.11 dependency file
a0ec861c9e157908d06a814079abbb63e54784e3 ripsrc/gitignore.go  0 0 0 0 0 0 0 modified jhaynie@pinpt.com 2018-09-05 19:12:32 +0000 UTC true - fixed issue where the dot in the regexp was improperly being double escaped - improve the skip reason to be more specific - add a few test and benchmark cases - add go.sum for new go1.11 dependency file
a0ec861c9e157908d06a814079abbb63e54784e3 ripsrc/gitignore_test.go Go 6 0 46 1205 40 2 0 added jhaynie@pinpt.com 2018-09-05 19:12:32 +0000 UTC true - fixed issue where the dot in the regexp was improperly being double escaped - improve the skip reason to be more specific - add a few test and benchmark cases - add go.sum for new go1.11 dependency file
`
