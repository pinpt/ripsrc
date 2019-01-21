package fileinfo

import (
	"fmt"
	"strings"

	enry "gopkg.in/src-d/enry.v1"
)

type Process struct {
	checkFilePathCache map[string]string
}

func New() *Process {
	s := &Process{}
	s.checkFilePathCache = map[string]string{}
	return s
}

const (
	skipLanguageUnknown      = "Language was unknown"
	skipFileSize             = "File size was %dK which exceeds limit of %dK"
	skipMaxLinesExceeded     = "File has more than %d lines"
	skipMaxLineBytesExceeded = "File has a line width of %d which is greater than max of %d"
	skipConfigFile           = "File was a config file"
	skipDotFile              = "File was a dot file"
	skipBlacklisted          = "File was on an exclusion list"
	skipVendoredFile         = "File was a vendored file"
	skipLicense              = "File is a license file"
)

type InfoArgs struct {
	FilePath string
	Lines    [][]byte
	Content  []byte
}

type Info struct {
	Language   string
	License    *License
	SkipReason string
}

// maxFileSize controls the size of the overall file we will process before
// determining that it's not a human written source file (generated, etc)
// and skip it
const maxFileSize = 1000000

// maxLinePerFile controls how many lines of code (LOC) we will process before
// determining that it's not a human written source file (generated, etc)
// and skip it
const maxLinePerFile = 40000

// maxBytesPerLine controls the size of one line we will process before
// determining that it's not a human written source file (generated, etc)
// and skip it
const maxBytesPerLine = 1096

func (s *Process) GetInfo(args InfoArgs) (res Info, skipReason string) {
	fileSize := len(args.Content)

	if fileSize > maxFileSize {
		return res, fmt.Sprintf(skipFileSize, fileSize/1000, maxFileSize/1000)
	}

	if possibleLicense(args.FilePath) {
		l, err := detect(args.FilePath, args.Content)
		if err != nil {
			panic(err)
		}
		if l != nil {
			res.License = l
			return res, skipLicense
		}
	}

	if skip := s.checkFilePath(args.FilePath); skip != "" {
		return res, skip
	}

	if len(args.Lines) > maxLinePerFile {
		return res, fmt.Sprintf(skipMaxLinesExceeded, maxLinePerFile)
	}

	for _, line := range args.Lines {
		if len(line) > maxBytesPerLine {
			return res, fmt.Sprintf(skipMaxLineBytesExceeded, len(line), maxBytesPerLine)
		}
	}

	res.Language = enry.GetLanguage(args.FilePath, args.Content)
	if res.Language == "" {
		return res, skipLanguageUnknown
	}

	return res, ""
}

func (s *Process) checkFilePath(filePath string) (skipReason string) {
	if res, ok := s.checkFilePathCache[filePath]; ok {
		return res
	}
	res := s.checkFilePathUncached(filePath)
	s.checkFilePathCache[filePath] = res
	return res
}

func (s *Process) checkFilePathUncached(filePath string) (skipReason string) {
	if enry.IsConfiguration(filePath) {
		return skipConfigFile
	}
	if enry.IsDotFile(filePath) {
		return skipDotFile
	}
	if ignorePatterns.MatchString(filePath) {
		return skipBlacklisted
	}
	if s.isVendored(filePath) {
		return skipVendoredFile
	}
	return ""
}

func (p *Process) isVendored(filePath string) bool {
	if enry.IsVendor(filePath) {
		// enry will incorrectly match something like:
		// src/com/foo/android/cache/DiskLruCache.java
		// as a vendored file but it's not.... we'll try
		// and correct with heuristics here
		if strings.HasPrefix(filePath, "src/") {
			return false
		}
		return true
	}
	return false
}
