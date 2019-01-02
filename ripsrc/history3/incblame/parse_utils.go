package incblame

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func startsWith(b []byte, prefix string) bool {
	if len(prefix) > len(b) {
		return false
	}
	return string(b[:len(prefix)]) == prefix
}

func parseContext(b0 []byte) (res []HunkLocation) {
	rerr := func(msg string) {
		panic(fmt.Errorf("invalid diff context format %v %v", string(b0), msg))
	}
	atc := 0
	for _, b := range b0 {
		if b == '@' {
			atc++
		} else {
			break
		}
	}
	endInd := 0
	c := 0
	for i, b := range b0 {
		if b == '@' {
			c++
		}
		if c == atc*2 {
			endInd = i
			break
		}
	}
	if endInd == 0 {
		rerr("endInd == 0")
	}
	b := b0[0:endInd] // remove section heading
	b = bytes.Trim(b, "@ ")
	parts := bytes.Split(b, []byte(" "))
	for _, p := range parts {
		if len(p) < 2 {
			rerr("")
		}
		one := HunkLocation{}
		switch p[0] {
		case '-':
			one.Op = OpDel
		case '+':
			one.Op = OpAdd
		default:
			rerr("invalid op")
		}
		parts := strings.Split(string(p[1:]), ",")
		if len(parts) != 2 && len(parts) != 1 {
			rerr("")
		}
		if len(parts) == 1 {
			var err error
			one.Lines, err = strconv.Atoi(parts[0])
			if err != nil {
				rerr("")
			}
			res = append(res, one)
			continue
		}
		var err error
		one.Offset, err = strconv.Atoi(parts[0])
		if err != nil {
			rerr("")
		}
		one.Lines, err = strconv.Atoi(parts[1])
		if err != nil {
			rerr("")
		}
		res = append(res, one)
	}
	return
}

const diffDeclPrefixNormal = "diff --git "
const diffDeclPrefixMerge = "diff --combined "

var errParseDiffDeclRenameWithSpaces = errors.New("can't parse diff declaration containing rename with spaces, extract data from rename line instead")

var errParseDiffDeclMerge = errors.New("merge diff decl does not contain all names, use diff instead")

func parseDiffDecl(diff []byte) (fromPath string, toPath string, _ error) {
	if len(diff) == 0 {
		return "", "", errors.New("empty string passed to parseDiffDecl")
	}
	if !startsWith(diff, diffDeclPrefixNormal) {
		if startsWith(diff, diffDeclPrefixMerge) {
			return "", "", errParseDiffDeclMerge
		}
		return "", "", fmt.Errorf("invalid prefix for diff decl %s", diff)
	}
	data := diff[len(diffDeclPrefixNormal):]
	spaceCount := countByte(data, ' ')
	if spaceCount == 0 {
		return "", "", fmt.Errorf("invalid format for diff decl, no space sep %s", diff)
	}
	remPrefix := func(data []byte, pr string) (string, error) {
		if len(data) <= len(pr) || string(data[0:len(pr)]) != pr {
			return "", fmt.Errorf("invalid format for diff decl %s, removing prefix %s", diff, data)
		}
		return string(data[len(pr):]), nil
	}
	// simple case with no spaces in name
	if spaceCount == 1 {
		i := bytes.IndexByte(data, ' ')
		if i == 0 || i == len(data) {
			return "", "", fmt.Errorf("invalid format for diff decl %s", diff)
		}
		var err error
		fromPath, err = remPrefix(data[0:i], "a/")
		if err != nil {
			return "", "", err
		}
		toPath, err = remPrefix(data[i+1:], "b/")
		if err != nil {
			return "", "", err
		}
		return
	}
	// only supporting space in name when from and to is the same, otherwise will have to rely on renames in parser
	mid := len(data) / 2
	if len(data) < 7 || len(data)%2 != 1 || string(data[0:2]) != "a/" || string(data[mid+1:mid+3]) != "b/" {
		//panic(fmt.Errorf("only supporting space in name when from and to is the same, otherwise need to handle this case separately, decl %s, prefixes %s:%s", diff, data[0:2], data[mid+1:mid+3]))
		return "", "", errParseDiffDeclRenameWithSpaces
	}

	fromPath = string(data[2:mid])
	toPath = string(data[mid+3:])
	if fromPath != toPath {
		return "", "", errParseDiffDeclRenameWithSpaces
		//panic(fmt.Errorf("fromPath != toPath, only supporting space in name when from and to is the same, otherwise need to handle this case separately, decl %s, fromPath:%v toPath:%v", diff, fromPath, toPath))
	}
	return
}

func countByte(data []byte, b byte) (res int) {
	for _, v := range data {
		if v == b {
			res++
		}
	}
	return
}
