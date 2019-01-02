package gitblame2

import (
	"reflect"
	"testing"
)

func TestParseOutput(t *testing.T) {
	data := `b4dadc54e312e976694161c2ac59ab76feb0c40d 1 1 2
author User1
author-mail <user1@example.com>
author-time 1543352136
author-tz +0100
committer User1
committer-mail <user1@example.com>
committer-time 1543352136
committer-tz +0100
summary c1
boundary
filename main.go
	package main
b4dadc54e312e976694161c2ac59ab76feb0c40d 2 2
	
b4dadc54e312e976694161c2ac59ab76feb0c40d 5 3 1
	func main() {
69ba50fff990c169f80de96674919033a0a9b66d 4 4 1
author User2
author-mail <user2@example.com>
author-time 1543352171
author-tz +0100
committer User2
committer-mail <user2@example.com>
committer-time 1543352171
committer-tz +0100
summary c2
previous b4dadc54e312e976694161c2ac59ab76feb0c40d main.go
filename main.go
		// do nothing
b4dadc54e312e976694161c2ac59ab76feb0c40d 7 5 2
	}
b4dadc54e312e976694161c2ac59ab76feb0c40d 8 6
	`

	got := parseOutput(data)

	c1hash := "b4dadc54e312e976694161c2ac59ab76feb0c40d"

	c1 := map[string]string{
		"author":         "User1",
		"author-mail":    "<user1@example.com>",
		"author-time":    "1543352136",
		"author-tz":      "+0100",
		"committer":      "User1",
		"committer-mail": "<user1@example.com>",
		"committer-time": "1543352136",
		"committer-tz":   "+0100",
		"summary":        "c1",
		"boundary":       "",
		"filename":       "main.go",
	}

	c2hash := "69ba50fff990c169f80de96674919033a0a9b66d"

	c2 := map[string]string{
		"author":         "User2",
		"author-mail":    "<user2@example.com>",
		"author-time":    "1543352171",
		"author-tz":      "+0100",
		"committer":      "User2",
		"committer-mail": "<user2@example.com>",
		"committer-time": "1543352171",
		"committer-tz":   "+0100",
		"summary":        "c2",
		"previous":       "b4dadc54e312e976694161c2ac59ab76feb0c40d main.go",
		"filename":       "main.go",
	}

	want := []line{
		{c1hash, "package main", c1},
		{c1hash, "", c1},
		{c1hash, "func main() {", c1},
		{c2hash, "	// do nothing", c2},
		{c1hash, "}", c1},
		{c1hash, "", c1},
	}

	assertEqualParsed(t, want, got)
}

func assertEqualParsed(t *testing.T, want, got []line) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("got invalid number of lines\n%+v", got)
	}
	for i := range want {
		if !reflect.DeepEqual(want[i], got[i]) {
			t.Fatalf("line %v does not match, got %+v", i, got[i])
		}
	}
}
