package ripsrc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiffParser(t *testing.T) {
	assert := assert.New(t)
	p := newDiffParser("ripsrc/git.go")
	ok, err := p.parse("diff --git a/ripsrc/git.go b/ripsrc/git.go")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(lookingForCommit, p.state)
	p.reset()
	ok, err = p.parse("diff --git a/.github/PULL_REQUEST_TEMPLATE.md b/.github/PULL_REQUEST_TEMPLATE.md")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(lookingForCommit, p.state)
	ok, err = p.parse("new file mode 100644")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(lookingForCommit, p.state)
	ok, err = p.parse("index 0000000..9d67477")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(lookingForCommit, p.state)
	ok, err = p.parse("--- /dev/null")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(lookingForCommit, p.state)
	ok, err = p.parse("+++ b/.github/PULL_REQUEST_TEMPLATE.md")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(lookingForCommit, p.state)
	p.reset()
	ok, err = p.parse("!SHA: 123456789")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(lookingForHeader, p.state)
	ok, err = p.parse("diff --git a/ripsrc/git.go b/ripsrc/git.go")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(lookingForStart, p.state)
	ok, err = p.parse("@@ -0,0 +1,5 @@")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(insidePatch, p.state)
	ok, err = p.parse("+**JIRA:** https://pinpt-hq.atlassian.net/browse/BE-XXX")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(insidePatch, p.state)
	ok, err = p.parse("+")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(insidePatch, p.state)
	ok, err = p.parse("+Provide a clear PR title prefixed with BE-XXX")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(insidePatch, p.state)
	ok, err = p.parse("+")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(insidePatch, p.state)
	ok, err = p.parse("+**Description:**")
	assert.True(ok)
	assert.NoError(err)
	assert.Equal(insidePatch, p.state)
	assert.NotNil(p.patch)
	assert.Len(p.history, 1)
	assert.Equal(`@@ -0,0 +1,5 @@
+**JIRA:** https://pinpt-hq.atlassian.net/browse/BE-XXX
+
+Provide a clear PR title prefixed with BE-XXX
+
+**Description:**
`, p.patchbuf.String())
	assert.NoError(p.complete())
	assert.Len(p.history, 1)
	assert.Equal(`@@ -0,0 +1,5 @@
+**JIRA:** https://pinpt-hq.atlassian.net/browse/BE-XXX
+
+Provide a clear PR title prefixed with BE-XXX
+
+**Description:**
`, p.history[0].Patch.String())
	assert.Equal(p.patch.String(), p.history[0].Patch.String())
}
