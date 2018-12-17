package diff

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

func parseContext(b []byte) (res []HunkContext) {
	rerr := func(msg string) {
		panic(fmt.Errorf("invalid diff context format %v %v", string(b), msg))
	}
	i := bytes.LastIndexByte(b, '@')
	b = b[0:i] // remove section heading
	b = bytes.Trim(b, "@ ")
	parts := bytes.Split(b, []byte(" "))
	for _, p := range parts {
		if len(p) < 4 {
			rerr("")
		}
		one := HunkContext{}
		switch p[0] {
		case '-':
			one.Op = OpDel
		case '+':
			one.Op = OpAdd
		default:
			rerr("invalid op")
		}
		parts := strings.Split(string(p[1:]), ",")
		if len(parts) != 2 {
			rerr("")
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
