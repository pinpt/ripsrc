package fileinfo

import "regexp"

// License holds details about detected license
type License struct {
	Name       string
	Confidence float32
}

var licenses = regexp.MustCompile("\\/?(LICENSE|LICENCE|README|COPYING|LICENSE-.*|UNLICENSE|UNLICENCE)(\\.(md|txt))?$")

func possibleLicense(filename string) bool {
	return licenses.MatchString(filename)
}
