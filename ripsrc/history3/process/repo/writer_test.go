package repo

import (
	"math/rand"
	"os"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
)

func randomBytes(c int) []byte {
	res := make([]byte, c)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	return res
}

func randomString(c int) string {
	return string(randomBytes(c))
}

func randomBlame(lines int) *incblame.Blame {
	res := &incblame.Blame{}
	res.Commit = randomString(32)
	l := randomBytes(1000)
	for i := 0; i < lines; i++ {
		res.Lines = append(res.Lines, &incblame.Line{
			Commit: randomString(32),
			Line:   l,
		})
	}
	return res
}

func BenchmarkWritingCheckpointRandomData(b *testing.B) {
	dir := tempDir()
	defer os.RemoveAll(dir)

	repo := New()

	for i := 0; i < 10; i++ {
		ch := randomString(32)
		repo.AddCommit(ch)
		for i := 0; i < 10; i++ {
			fp := randomString(100)
			repo[ch][fp] = randomBlame(100)
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := WriteCheckpoint(repo, dir)
		if err != nil {
			b.Fatal(err)
		}
	}
}
