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

func parseContext(b []byte) (res []HunkLocation) {
	rerr := func(msg string) {
		panic(fmt.Errorf("invalid diff context format %v %v", string(b), msg))
	}
	i := bytes.LastIndexByte(b, '@')
	b = b[0:i] // remove section heading
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