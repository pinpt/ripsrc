package tests

import (
	"testing"
)

func TestMultipleBranches1(t *testing.T) {
	tc := NewTest(t, "basic",
		nil)
	br := tc.Run()

	c1 := "faeab7d021e1f98c74831b9472ad8616f60fe8d1"
	c2 := "8c87905b207af8222a2c41e462488b66d9be5057"
	c3 := "ad31357538f995eedada0d59654c80d16ab67e7a"

	want := map[string][]string{
		c1: []string{"b", "master"},
		c2: []string{"master"},
		c3: []string{"b"},
	}

	assertResult(t, br, want)
}
