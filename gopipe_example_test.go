package gopipe_test

import (
	"log"

	. "github.com/urjitbhatia/gopipe"
)

type ExamplePipe struct{}

func (ep ExamplePipe) Process(in chan interface{}, out chan interface{}) {
	for item := range in {
		if intval, ok := item.(int); ok {
			out <- intval * 2
		} else {
			log.Println("not ok")
		}
	}
}

func ExamplePipeline() {
	max := 20
	ep := ExamplePipe{}
	sp := subtractingPipe{}
	pipeline := NewPipeline(ep, sp)

	pipeinput := intGenerator(max)
	pipeline.AttachSource(pipeinput)

	pipeout := make(chan interface{})
	pipeline.AttachSink(pipeout)

	for i := 0; i < max; i += 1 {
		log.Println("value is:", <-pipeout)
	}
}
