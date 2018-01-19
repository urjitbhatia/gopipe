package gopipe_test

import (
	"log"

	. "github.com/urjitbhatia/gopipe"
)

type ExamplePipe struct{}

func (ep ExamplePipe) Process(in chan interface{}, out chan interface{}) {
	for {
		select {
		case item, more := <-in:
			if !more {
				log.Println("Pipe-in closed")
				close(out)
				return
			}
			if intval, ok := item.(int); ok {
				out <- intval * 2
			} else {
				log.Println("not ok")
			}
		}
	}
}

func ExamplePipeline() {
	ep := ExamplePipe{}
	sp := subtractingPipe{}
	pipeline := NewPipeline(ep, sp)

	pipeinput := intGenerator(20)
	pipeline.AttachSource(pipeinput)

	pipeout := make(chan interface{})
	pipeline.AttachSink(pipeout)

	var start = 0
outloop:
	for {
		select {
		case val, more := <-pipeout:
			if !more {
				pipeout = nil
				break outloop
			}
			log.Println("value is:", val)
			start++
		}
	}
}
