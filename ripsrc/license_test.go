package ripsrc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
