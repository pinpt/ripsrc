package logger

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

type Logger interface {
	Info(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

type DefaultLogger struct {
	wr io.Writer
	mu sync.Mutex
}

func NewDefaultLogger(wr io.Writer) Logger {
	s := &DefaultLogger{}
	s.wr = wr
	return s
}

func (s *DefaultLogger) Info(msg string, args ...interface{}) {
	s.log("INFO", msg, args...)
}

func (s *DefaultLogger) Debug(msg string, args ...interface{}) {
	s.log("DEBUG", msg, args...)
}

func (s *DefaultLogger) log(kind string, msg string, args ...interface{}) {
	write := func(format string, args ...interface{}) {
		s.mu.Lock()
		defer s.mu.Unlock()
		p := fmt.Sprintf(format, args...)
		_, err := s.wr.Write([]byte(p))
		if err != nil {
			panic(err)
		}
		_, err = s.wr.Write([]byte("\n"))
		if err != nil {
			panic(err)
		}
	}
	kvs, err := formatArgs(args)
	if err != nil {
		write("ERROR Logger invalid args passed. Msg: %v Args: %v Err: %v", msg, args, err)
	}
	write("%v %v %v", kind, msg, kvs)
}

type kv struct {
	K string
	V string
}

func formatArgs(args []interface{}) (res []kv, _ error) {
	if len(args)%2 != 0 {
		return nil, errors.New("len of args not even")
	}
	for i := 0; i < len(args); i += 2 {
		k, ok := args[i].(string)
		if !ok {
			return nil, errors.New("key arg passes in not a string")
		}
		v := fmt.Sprintf("%v", args[i+1])
		res = append(res, kv{k, v})
	}
	return
}
