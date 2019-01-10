func makeLongDiffCreate(c int) Diff {
	diffPrefix := `diff --git a/a.txt b/a.txt
+++ b/a.txt
func TestApplyNewGenFile(t *testing.T) {
	diff := makeLongDiffCreate(lines)
	diff := makeLongDiffCreate(lines)

func makeLongDiffRemoval(c int) Diff {
	cstr := strconv.Itoa(c)
	diffPrefix := `diff --git a/a.txt b/a.txt	
--- /dev/null
+++ b/a.txt
@@ -1,` + cstr + ` +1 @@` + "\n" +
		" a\n"

	diffBytes := []byte{}
	diffBytes = append(diffBytes, diffPrefix...)
	for i := 0; i < c-1; i++ {
		diffBytes = append(diffBytes, "-a"...)
		diffBytes = append(diffBytes, '\n')
	}

	return Parse(diffBytes)
}

func TestApplyRemovalsGen(t *testing.T) {
	const lines = 3
	diff1 := makeLongDiffCreate(lines)
	f1 := Apply(Blame{}, diff1, "c1", "")
	diff2 := makeLongDiffRemoval(lines)
	f2 := Apply(f1, diff2, "c2", "")
	want := Blame{
		Lines: []Line{
			tl("a", "c1"),
		},
	}
	assertEqualFiles(t, f2, want)
}

func BenchmarkApplyLargeRemoval(b *testing.B) {
	const lines = 10000
	diff1 := makeLongDiffCreate(lines)
	f1 := Apply(Blame{}, diff1, "c1", "")
	diff2 := makeLongDiffRemoval(lines)

	for i := 0; i < b.N; i++ {
		Apply(f1, diff2, "c1", "")
	}
}

func makeLongDiffAdd(c int) Diff {
	cstr := strconv.Itoa(c)
	diffPrefix := `diff --git a/a.txt b/a.txt	
--- /dev/null
+++ b/a.txt
@@ -1 +1,` + cstr + `@@` + "\n"

	res := []byte{}
	res = append(res, diffPrefix...)
	for i := 0; i < c; i++ {
		res = append(res, "+b"...)
		res = append(res, '\n')
	}

	res = append(res, " a\n"...)

	return Parse(res)
}

func TestApplyAdditionsStartGen(t *testing.T) {
	const lines = 3
	diff1 := makeLongDiffCreate(1)
	f1 := Apply(Blame{}, diff1, "c1", "")
	diff2 := makeLongDiffAdd(lines)
	f2 := Apply(f1, diff2, "c2", "")
	want := Blame{
		Lines: []Line{
			tl("b", "c2"),
			tl("b", "c2"),
			tl("b", "c2"),
			tl("a", "c1"),
		},
	}
	assertEqualFiles(t, f2, want)
}

func BenchmarkApplyLargeAdditionsStart(b *testing.B) {
	const lines = 10000
	diff1 := makeLongDiffCreate(1)
	f1 := Apply(Blame{}, diff1, "c1", "")
	diff2 := makeLongDiffAdd(lines)

	for i := 0; i < b.N; i++ {
		Apply(f1, diff2, "c2", "")
	}
}