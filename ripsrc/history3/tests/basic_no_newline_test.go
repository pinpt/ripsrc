package tests

import (
	"testing"

	"github.com/pinpt/ripsrc/ripsrc/history3/incblame"
	"github.com/pinpt/ripsrc/ripsrc/history3/process"
)

func TestBasicNoNewline(t *testing.T) {
	test := NewTest(t, "basic_no_newline")
	got := test.Run()

	c1 := "520bd1198f9640c56e5245c60ed920364770452b"
	c2 := "a4bb5c8a7a31319078ad20e809b4d195ec26d5f4"
	c3 := "2a132254cf48a2633d152795fc1b026356580634"

	want := []process.Result{
		{
			Commit: c1,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c1,
					line(`a`, c1),
				),
			},
		},
		{
			Commit: c2,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c2,
					line(`a`, c2),
				),
			},
		},
		{
			Commit: c3,
			Files: map[string]*incblame.Blame{
				"a.txt": file(c3,
					line(`a`, c3),
				),
			},
		},
	}
	assertResult(t, want, got)
}
