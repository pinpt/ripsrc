package repo

import (
	"os"
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/pkg/random"
)

func randomString(c int) string {
	return random.String(c, random.LatinAndNumbers)
}

func randomBlame(lines int) *incblame.Blame {
	return randomBlameLineLen(lines, 1000)
}

func randomBlameLineLen(lines int, lineLen int) *incblame.Blame {
	res := &incblame.Blame{}
	res.Commit = randomString(32)
	l := randomString(lineLen)
	for i := 0; i < lines; i++ {
		res.Lines = append(res.Lines, &incblame.Line{
			Commit: randomString(32),
			Line:   []byte(l),
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
		err := WriteCheckpoint(repo, dir, "c1")
		if err != nil {
			b.Fatal(err)
		}
	}
}
