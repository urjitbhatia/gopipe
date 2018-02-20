package gopipe

import (
	"fmt"
	"log"
	"time"
)

/*
Pipe is a single component that processes items. Pipes can be composed to form a pipeline
*/
type Pipe interface {
	Process(in chan interface{}, out chan interface{})
}

/*
Pipeline connects multiple pipes in order. The head chan receives incoming items
and tail chan send out items that the pipeline has finished processing.
*/
type Pipeline struct {
	bufferSize int              // Chan buffer size
	head       chan interface{} // Chan representing the head of the pipeline
	tail       chan interface{} // Chan representing the tail of the pipeline
	debugLog   bool             // Option to log pipeline state transitions, false by default
}

/*
Enqueue takes an item one at a time and adds it to the start of the pipeline.
Use AttachSource to attach a chan of incoming items to the pipeline.
If the pipeline is blocked, this is going to be a blocking operation
*/
func (p *Pipeline) Enqueue(item interface{}) {
	p.head <- item
}

/*
Dequeue will block till an item is available and then dequeue from the pipeline.
*/
func (p *Pipeline) Dequeue() interface{} {
	return <-p.tail
}

/*
DequeueTimeout will block till an item is available and then dequeue from the pipeline.
*/
func (p *Pipeline) DequeueTimeout(t time.Duration) interface{} {
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(t)
		timeout <- true
	}()
	select {
	case item := <-p.tail:
		// a read from tail has occurred
		return item
	case <-timeout:
		// the read has timed out
		return nil
	}
}

/*
Close makes sure that the pipeline accepts no further messages.
If the go routine/method writing to the pipeline is still Enqueuing, it will
cause a panic - can't write to a closed channel
*/
func (p *Pipeline) Close() {
	close(p.head)
}

// String prints a helpful debug state of the pipeline
func (p *Pipeline) String() {
	log.Printf("BufferSize: %d Head: %v Tail: %v DebugMode: %t", p.bufferSize, p.head,
		p.tail, p.debugLog)
}

/*
AddPipe attaches a pipe to the end of the pipeline.
This will immediately start routing items to this newly attached pipe
*/
func (p *Pipeline) AddPipe(pipe Pipe) *Pipeline {
	oldTail := p.tail
	newTail := make(chan interface{}, p.bufferSize)
	go pipe.Process(oldTail, newTail)
	p.tail = newTail
	return p
}

func (p *Pipeline) AddJunction(fn RoutingFunc) *Junction {
	j := Junction{routingFn: fn, router: make(map[interface{}]*Pipeline)}
	j.in = p.tail
	go func() {
		for item := range j.in {
			routingKey := fn(item)
			p.debug(fmt.Sprintf("routing key: %v", routingKey))
			if dest, ok := j.router[routingKey]; ok {
				dest.Enqueue(item)
			} else {
				p.debug(fmt.Sprintf("Junction discarding item: %+v no dest pipeline for routing key: %+v",
					item, routingKey))
			}
		}
	}()
	return &j
}

/*
AttachSource accepts the source channel as the entry point to the pipeline
*/
func (p *Pipeline) AttachSource(source chan interface{}) {
	p.debug("Attaching source channel to pipeline")
	go func() {
		for item := range source {
			p.head <- item
		}
		p.debug("Pipeline source closed. Closing rest of the pipeline")
		p.Close()
	}()
}

/*
AttachTap adds a Tap to the pipeline.
*/
func (p *Pipeline) AttachTap(tapOut chan interface{}) {
	p.debug("Attaching tap to pipeline")

	// Make a new tail for this Pipeline
	tapTail := p.tail
	p.tail = make(chan interface{})
	tap := make(chan interface{})
	go func() {
		// read from old tail and send on local tap chan as well as pipeline tail
		for item := range tapTail {
			tap <- item
			p.tail <- item
		}
		p.debug("Pipeline tap source closed. Closing rest of the pipeline")
		close(tap)
		close(p.tail)
	}()

	go func() {
		// routine that consumes from local tap and queues items at external tap
		for item := range tap {
			tapOut <- item
		}
		close(tapOut)
	}()
}

/*
AttachSink takes a terminating channel and dequeues the messages
from the pipeline onto that channel.
*/
func (p *Pipeline) AttachSink(out chan interface{}) {
	p.debug("Attaching sink to pipeline")
	go func() {
		for item := range p.tail {
			out <- item
		}
		p.debug("Shutting down pipeline Sink")
		close(out)
	}()
}

/*
AttachSinkFanOut redirects outgoing items to the appropriate channel based on the routing function provided.
Returns a channel where unrouted items are pushed. If the routing function returns a routing key that does not have an associated
channel provided, the item will be routed to the unrouted channel. Items encountering errors on routing are also put on the unrouted
channel. Clients of the library should handle unrouted chan properly - if nothing is listening on that chan, operations will block if
an unroutable item is put on the channel (or until its buffer is full)
*/
func (p *Pipeline) AttachSinkFanOut(chanfan map[string]chan interface{}, unrouted chan interface{}, routingFunc func(interface{}) (string, error)) {
	go func() {
		for item := range p.tail {
			key, err := routingFunc(item)
			routeChan, ok := chanfan[key]
			if err != nil || key == "" || !ok {
				routeChan = unrouted
			}
			routeChan <- item
		}
		p.debug("Shutting down Pipeline ChanFans")
		for key, fanoutChan := range chanfan {
			// Close all outgoing channels
			p.debug("Closing channel for routing key:", key)
			close(fanoutChan)
		}
		close(unrouted)
	}()
}

/*
debug Prints log statements if debugLog is true
*/
func (p *Pipeline) debug(values ...string) {
	if p.debugLog {
		log.Println(values)
	}
}

/*
Debug enables logging on this pipeline
*/
func (p *Pipeline) Debug() {
	p.debugLog = true
}

/*
NewBufferedPipeline creates a Pipeline with channel buffers set to the given size.
This is useful in increasing processing speed. NewPipeline should mostly always
be tried first.
*/
func NewBufferedPipeline(s int, pipes ...Pipe) *Pipeline {
	if len(pipes) == 0 {
		// Without pipes, just join head and tail
		h := make(chan interface{}, s)
		return &Pipeline{
			bufferSize: s,
			head:       h,
			tail:       h,
		}
	}
	var tail chan interface{}

	globalHead := make(chan interface{}, s)
	head := globalHead
	for _, pipe := range pipes {
		tail = make(chan interface{}, s)
		go pipe.Process(head, tail)
		head = tail
	}
	return &Pipeline{bufferSize: s, head: globalHead, tail: tail}
}

/*
NewPipeline takes multiple pipes in-order and connects them to form a pipeline.
Enqueue and Dequeue methods are used to attach source/sink to the pipeline.
If debugLog is true, logs state transitions to stdout.
*/
func NewPipeline(pipes ...Pipe) *Pipeline {
	return NewBufferedPipeline(0, pipes...)
}
