package fileinfo

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLicenseMatcher(t *testing.T) {
	assert := assert.New(t)
	assert.True(possibleLicense("/LICENSE.txt"))
	assert.True(possibleLicense("LICENSE.txt"))
	assert.True(possibleLicense("/cmd/foo/LICENSE"))
	assert.True(possibleLicense("LICENCE"))
	assert.True(possibleLicense("LICENCE.md"))
	assert.True(possibleLicense("LICENCE.txt"))
	assert.True(possibleLicense("LICENSE"))
	assert.True(possibleLicense("README"))
	assert.True(possibleLicense("README.md"))
	assert.True(possibleLicense("README.txt"))
	assert.True(possibleLicense("/cmd/README.txt"))
	assert.True(possibleLicense("UNLICENSE"))
	assert.True(possibleLicense("COPYING"))
	assert.True(possibleLicense("LICENSE-MIT"))
	assert.True(possibleLicense("LICENSE-MIT.md"))
	assert.True(possibleLicense("cmd/foo/LICENSE-MIT.txt"))
	assert.False(possibleLicense("main.go"))
}

func TestLicenseDetection(t *testing.T) {
	assert := assert.New(t)
	lic, err := detect("LICENSE", []byte(`Copyright 2018 Pinpoint

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.`))
	assert.NoError(err)
	assert.NotNil(lic)
	assert.Equal("Apache-2.0", lic.Name)
	assert.Equal(float32(1.0), lic.Confidence)
}

func TestLicenseDetectionNotFound(t *testing.T) {
	assert := assert.New(t)
	lic, err := detect("foo.go", []byte(`package foo
func main() {
}`))
	assert.NoError(err)
	assert.Nil(lic)
}

func TestLicenseMITFilename(t *testing.T) {
	assert := assert.New(t)
	lic, err := detect("LICENSE-MIT", []byte(`Copyright 2018 Pinpoint

	Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.`))
	assert.NoError(err)
	assert.NotNil(lic)
	assert.Equal("MIT", lic.Name)
	assert.Equal(float32(0.9814815), lic.Confidence)
}

func TestLicenseMITFilenameInDirectory(t *testing.T) {
	assert := assert.New(t)
	lic, err := detect("this/is/some/directory/license.txt", []byte(`Copyright 2018 Pinpoint

	Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.`))
	assert.NoError(err)
	assert.NotNil(lic)
	assert.Equal("MIT", lic.Name)
	assert.Equal(float32(0.9814815), lic.Confidence)
}

func TestLicenseConcurrency(t *testing.T) {
	assert := assert.New(t)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lic, err := detect("this/is/some/directory/license.txt", []byte(`Copyright 2018 Pinpoint

			Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

			The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

			THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.`))
			assert.NoError(err)
			assert.NotNil(lic)
			assert.Equal("MIT", lic.Name)
			assert.Equal(float32(0.9814815), lic.Confidence)
		}()
	}
	wg.Wait()
}
