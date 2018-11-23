package ripsrc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitIgnorePatterns(t *testing.T) {
	assert := assert.New(t)
	assert.True(ignorePatterns.MatchString("go.mod"))
	assert.True(ignorePatterns.MatchString("go.sum"))
	assert.True(ignorePatterns.MatchString("foo/go.sum"))
	assert.False(ignorePatterns.MatchString("foogo.mod"))
	assert.False(ignorePatterns.MatchString("foogo.sum"))
}

func TestShouldIgnore(t *testing.T) {
	assert := assert.New(t)
	b := &BlameProcessor{
		hashedExclusions: make(map[string]*exclusionDecision),
	}
	ok, reason := b.shouldProcess("go.mod")
	assert.False(ok)
	assert.Equal("file was on an exclusion list", reason)
	ok, reason = b.shouldProcess(".foo")
	assert.False(ok)
	assert.Equal("file was a dot file", reason)
	ok, reason = b.shouldProcess("vendor/foo/bar.go")
	assert.False(ok)
	assert.Equal("file was on an exclusion list", reason)
}

func BenchmarkIgnorePatterns10(b *testing.B) {
	assert := assert.New(b)
	for n := 0; n < b.N; n++ {
		assert.True(ignorePatterns.MatchString("go.mod"))
	}
}

func BenchmarkIgnore10(b *testing.B) {
	assert := assert.New(b)
	w := &BlameProcessor{
		hashedExclusions: make(map[string]*exclusionDecision),
	}
	for n := 0; n < b.N; n++ {
		ok, _ := w.shouldProcess("go.mod")
		assert.False(ok)
	}
}
