package kanikousen

import (
	"fmt"
)

type Broaker interface {
	FetchLoop(chan *Job, *Schedule)
	Close()
	Failed(*Job, error)
	Finish(*Job, int)
}

type Schedule struct {
	Queues    []string
	JobsCount uint
}

type Job struct {
	Id     uint
	TaskId string
	Args   []string
	Data   interface{}
}

type BroakerFactory func(*Boat, map[string]string) (Broaker, error)

var BroakerImplMap = make(map[string]BroakerFactory)

func RegisterBroaker(id string, factory BroakerFactory) {
	BroakerImplMap[id] = factory
}

func BroakerFromSpec(boat *Boat, spec string) (Broaker, error) {
	impl, opts, err := ParseSpecOpts(spec)
	if err != nil {
		return nil, fmt.Errorf("incorrect broaker spec: %s", err)
	}
	factory, ok := BroakerImplMap[impl]
	if !ok {
		return nil, fmt.Errorf("No such driver avaiable: %s", impl)
	}
	return factory(boat, opts)
}
