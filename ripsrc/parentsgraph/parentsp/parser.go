package parentsp

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

type Parser struct {
	r io.Reader
}

func New(r io.Reader) *Parser {
	p := &Parser{}
	p.r = r
	return p
}

type Parents map[string][]string

const mb = 1000 * 1000
const maxLine = 1 * mb

func (s *Parser) Run() (Parents, error) {
	res := Parents{}

	scanner := bufio.NewScanner(s.r)
	scanner.Buffer(nil, maxLine)
	for scanner.Scan() {
		line := scanner.Bytes()
		parts := bytes.Split(line, []byte("@"))

		commit := string(parts[0])
		var parents []string
		if len(parts[1]) != 0 {
			parents = strings.Split(string(parts[1]), " ")
		}
		res[commit] = parents
	}
	if err := scanner.Err(); err != nil {
		return res, err
	}
	return res, nil
}
