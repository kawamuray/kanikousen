package kanikousen

// This is logger module for "CONTAINER's STD{OUT,ERR}", not for this program logging

import (
	"fmt"
	"io"
)

type Logger interface {
	Yield(stdout, stderr io.Reader)
}

type LoggerFactory func(*Job, map[string]string) (Logger, error)

var LoggerImplMap = make(map[string]LoggerFactory)

func RegisterLogger(id string, factory LoggerFactory) {
	LoggerImplMap[id] = factory
}

func LoggerFactoryFromSpec(spec string) (func(*Job) (Logger, error), error) {
	impl, opts, err := ParseSpecOpts(spec)
	if err != nil {
		return nil, fmt.Errorf("incorrect logger spec: %s", err)
	}
	factory, ok := LoggerImplMap[impl]
	if !ok {
		return nil, fmt.Errorf("No such logger avaiable: %s", impl)
	}
	wrapper := func(job *Job) (Logger, error) {
		return factory(job, opts)
	}
	return wrapper, nil
}
