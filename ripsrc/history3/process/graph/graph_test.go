package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func str(s ...string) []string {
	return s
}

func TestLastCommonParents1(t *testing.T) {
	g := Graph{}
	g["c1"] = nil
	g["c2"] = str("c1")
	g["c3"] = str("c1")
	res := g.LastCommonParent(str("c2", "c3"))
	assert.Equal(t, "c1", res)
}

func TestLastCommonParents2(t *testing.T) {
	g := Graph{}
	g["c1"] = nil
	g["c2"] = str("c1")
	g["c3"] = str("c2")
	g["c4"] = str("c3")
	g["c5"] = str("c4")
	g["c6"] = str("c4")
	g["c7"] = str("c6")
	g["c8"] = str("c6")
	res := g.LastCommonParent(str("c5", "c8"))
	assert.Equal(t, "c4", res)
}
