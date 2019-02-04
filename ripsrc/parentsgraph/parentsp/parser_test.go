package parentsp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {

	data := `e99cb00954f08c1d33c5935742809868335483bf@
d497eccaf64c229771f471386cf49e4f653a00cb@e99cb00954f08c1d33c5935742809868335483bf`

	p := New(strings.NewReader(data))
	got, err := p.Run()
	if err != nil {
		t.Fatal(err)
	}
	want := Parents{
		"e99cb00954f08c1d33c5935742809868335483bf": nil,
		"d497eccaf64c229771f471386cf49e4f653a00cb": []string{
			"e99cb00954f08c1d33c5935742809868335483bf"},
	}

	assert.Equal(t, want, got)
}
func TestMerge(t *testing.T) {

	data := `f82b3491fbf1e4fd5666748efe0b198b82d587be@2fbc9d8afd98d677074ab2dc77658dbc2988e853 b7f8fa5c1794de8c7c36b61ba5e7e41e647ae97a`

	p := New(strings.NewReader(data))
	got, err := p.Run()
	if err != nil {
		t.Fatal(err)
	}
	want := Parents{
		"f82b3491fbf1e4fd5666748efe0b198b82d587be": []string{"2fbc9d8afd98d677074ab2dc77658dbc2988e853", "b7f8fa5c1794de8c7c36b61ba5e7e41e647ae97a"},
	}

	assert.Equal(t, want, got)
}
