package repo

/*
import (
	"fmt"
	"os"

	"github.com/tinylib/msgp/msgp"

	"github.com/pinpt/ripsrc/ripsrc/history3/process/repo/disk"

	"github.com/cespare/xxhash"
)

type store struct {
	data map[storeKey][]byte
}

type storeKey uint64

func newStore() *store {
	s := &store{}
	s.data = map[storeKey][]byte{}
	return s
}

const mb = 1000 * 1000

func newStoreFromFile(loc string) (*store, error) {
	s := newStore()
	f, err := os.Open(loc)
	if err != nil {
		return nil, err
	}
	r := msgp.NewReaderSize(f, 100*mb)
	var d disk.HashAndData
	for {
		err := d.DecodeMsg(r)
		if err != nil {
			// TODO: better check
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
		s.data[storeKey(d.Hash)] = d.Data
	}
	return s, nil
}

func (s *store) Serialize(loc string) error {
	f, err := os.Create(loc + ".tmp")
	if err != nil {
		return err
	}
	wr := msgp.NewWriter(f)
	for k, v := range s.data {
		d := disk.HashAndData{}
		d.Hash = uint64(k)
		d.Data = v
		err := d.EncodeMsg(wr)
		if err != nil {
			return err
		}
	}
	err = wr.Flush()
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return os.Rename(loc+".tmp", loc)
}

func (s *store) DataToKey(data []byte) storeKey {
	k := xxhash.Sum64(data)
	return storeKey(k)
}

func (s *store) Save(data []byte) (key storeKey) {

	key = s.DataToKey(data)

	s.data[key] = data
	return key
}

func (s *store) Get(key storeKey) []byte {
	res, ok := s.data[key]
	if !ok {
		panic(fmt.Errorf("data not found for key %v", key))
	}
	return res
}
*/
