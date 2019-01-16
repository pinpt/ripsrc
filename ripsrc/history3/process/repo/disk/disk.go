//go:generate msgp

package disk

type HashAndData struct {
	Hash uint64 `msg:"h"`
	Data []byte `msg:"d"`
}

type Commit struct {
	Files []string `msg:"f"`
}

type BlameData struct {
	IsBinary bool   `msg:"ib"`
	Lines    []Line `msg:"l"`
}

type Line struct {
	Commit  string `msg:"c"`
	DataKey uint64 `msg:"dk"`
}
