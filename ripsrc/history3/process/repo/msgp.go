package repo

import (
	"compress/gzip"
	"os"
	"path/filepath"

	"github.com/tinylib/msgp/msgp"
)

func msgpWriteToFile(loc string, obj msgp.Encodable) error {
	err := os.MkdirAll(filepath.Dir(loc), 0777)
	if err != nil {
		return err
	}
	f, err := os.Create(loc + ".tmp")
	if err != nil {
		return err
	}
	gw := gzip.NewWriter(f)
	wr := msgp.NewWriter(gw)
	err = obj.EncodeMsg(wr)
	if err != nil {
		return err
	}
	err = wr.Flush()
	if err != nil {
		return err
	}
	err = gw.Flush()
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return os.Rename(loc+".tmp", loc)
}

func msgpReadFromFile(loc string, obj msgp.Decodable) error {
	f, err := os.Open(loc)
	if err != nil {
		return err
	}
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	r := msgp.NewReader(gr)
	err = obj.DecodeMsg(r)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return nil
}
