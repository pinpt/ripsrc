// +build ignore

package main

/**
 * this generator will generate a regular expression using the
 * github/gitignore project which is how github decides which files
 * should be ignored based on numerous user contributed settings. we
 * use this large corpus to exclude files which normally should be ignored
 * in source repos
 */

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func findFiles(dir string, pattern *regexp.Regexp) ([]string, error) {
	fileList := []string{}
	err := filepath.Walk(dir, func(p string, f os.FileInfo, err error) error {
		// fmt.Println("trying to match", pattern, "for", path)
		if pattern.MatchString(filepath.Base(p)) {
			fileList = append(fileList, p)
		}
		return nil
	})
	return fileList, err
}

func format(line string) string {
	line = strings.Replace(line, ".", "\\.", -1)
	line = strings.Replace(line, "*", "(.*?)", -1)
	return line
}

func parseFile(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening %v. %v", filename, err)
	}
	defer f.Close()
	b := bufio.NewReader(f)
	matches := []string{}
	for {
		line, err := b.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		line = strings.TrimSpace(line[0 : len(line)-1])
		if line == "" || line == "\n" || line[0] == '#' || line[0] == '!' || line[0] == '\t' {
			continue
		}
		i := strings.Index(line, "#")
		if i > 0 {
			line = strings.TrimSpace(line[0:i])
		}
		matches = append(matches, format(line))
	}
	return matches, nil
}

func main() {
	var files []string
	files = append(files, filepath.Join("..", "custom_patterns.txt"))
	matchers := []string{}
	for _, filename := range files {
		patterns, err := parseFile(filename)
		if err != nil {
			panic(err)
		}
		matchers = append(matchers, patterns...)
	}
	regstr := "`(" + strings.Join(matchers, "|") + ")`"
	regexp.MustCompile(regstr) // make sure it compiles
	outfile, _ := filepath.Abs("gitignore.go")
	tmpl := fmt.Sprintf(`// DO NOT EDIT -- generated code

package ripsrc

import "regexp"

var ignorePatterns = regexp.MustCompile(%s)
`, regstr)
	ioutil.WriteFile(outfile, []byte(tmpl), 0644)
}
