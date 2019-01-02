package incblame

import (
	"bytes"
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
