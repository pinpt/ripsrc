package fileinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLicense1(t *testing.T) {
	p := New()
	info, skipReason := p.GetInfo(makeArgs("COPYING",
		`Copyright 2018 Pinpoint

	Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.`))
	assert.Equal(t, "File is a license file", skipReason)
	if info.License == nil {
		t.Fatal("failed to detect license")
	}
	if info.License.Name != "MIT" {
		t.Fatal("invalid license name")
	}
	if info.License.Confidence < 0.9 {
		t.Fatal("too lowe license condidense.")
	}
}
