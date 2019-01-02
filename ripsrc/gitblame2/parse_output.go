package gitblame2

import (
	"strings"
)

type line struct {
	CommitHash string
	Content    string
	Meta       map[string]string
}

func parseOutput(data string) (res []line) {
	lines := strings.Split(data, "\n")
	metasByCommit := map[string]map[string]string{}
	for i := 0; i < len(lines); {
		fl := lines[i]
		if fl == "" && i == len(lines)-1 {
			// skip last empty line
			break
		}
		parts := strings.Split(fl, " ")
		rl := line{}
		rl.CommitHash = parts[0]
		rl.Meta = map[string]string{}
		for {
			i++
			if i >= len(lines) {
				panic("after header in git blame we need the content line")
			}
			l := lines[i]
			if l[0] != '\t' {
				parts := strings.SplitN(l, " ", 2)
				if len(parts) == 2 {
					rl.Meta[parts[0]] = parts[1]
				} else {
					// i.e. boundary
					rl.Meta[l] = ""
				}
			} else {
				rl.Content = l[1:]
				break
			}
		}
		if len(rl.Meta) == 0 {
			rl.Meta = metasByCommit[rl.CommitHash]
		} else {
			metasByCommit[rl.CommitHash] = rl.Meta
		}
		res = append(res, rl)
		i++
	}
	return
}
