//go:generate msgp

package disk

type Data struct {
	Data     []DataRow  `msg:"data"`
	Blames   []Blame    `msg:"bl"`
	Lines    []Line     `msg:"l"`
	LineData []LineData `msg:"ld"`
}

type DataRow struct {
	Commit       string `msg:"c"`
	Path         string `msg:"p"`
	BlamePointer uint64 `msg:"bp"`
}

type Blame struct {
	Pointer      uint64   `msg:"p"`
	Commit       string   `msg:"c"`
	LinePointers []uint64 `msg:"lp"`
	IsBinary     bool     `msg:"ib"`
}

type Line struct {
	Pointer         uint64 `msg:"p"`
	Commit          string `msg:"c"`
	LineDataPointer uint64 `msg:"ldp"`
}

type LineData struct {
	Pointer uint64 `msg:"p"`
	Data    []byte `msg:"d"`
}
