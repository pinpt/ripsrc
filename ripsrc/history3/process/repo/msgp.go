package repo

import (
	"compress/gzip"
	"os"
	"path/filepath"

	"github.com/tinylib/msgp/msgp"
)

type msgWriter struct {
	loc string
	f   *os.File
	gw  *gzip.Writer
	wr  *msgp.Writer
}

func newMsgWriter(dir string, kind string) (*msgWriter, error) {
	s := &msgWriter{}
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return nil, err
	}
	s.loc = filepath.Join(dir, kind)
	f, err := os.Create(s.loc + ".tmp")
	if err != nil {
		return nil, err
	}
	s.f = f
	s.gw = gzip.NewWriter(f)
	s.wr = msgp.NewWriter(s.gw)
	return s, nil
}

func (s *msgWriter) Write(obj msgp.Encodable) error {
	return obj.EncodeMsg(s.wr)
}

func (s *msgWriter) Finish() error {
	err := s.wr.Flush()
	if err != nil {
		return err
	}
	err = s.gw.Flush()
	if err != nil {
		return err
	}
	err = s.f.Close()
	if err != nil {
		return err
	}
	return os.Rename(s.loc+".tmp", s.loc)
}

type msgReader struct {
	loc string
	f   *os.File
	gr  *gzip.Reader
	r   *msgp.Reader
}

func newMsgReader(dir string, kind string) (*msgReader, error) {
	s := &msgReader{}
	s.loc = filepath.Join(dir, kind)
	f, err := os.Open(s.loc)
	if err != nil {
		return nil, err
	}
	s.f = f
	s.gr, err = gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	s.r = msgp.NewReader(s.gr)
	return s, nil
}

func (s *msgReader) Read(obj msgp.Decodable) error {
	return obj.DecodeMsg(s.r)
}

func (s *msgReader) Finish() error {
	err := s.f.Close()
	if err != nil {
		return err
	}
	return nil
}
