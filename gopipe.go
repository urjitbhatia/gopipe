package gopipe

import (
	"log"
	"time"
)

/*
Pipe is a single component that processes items. Pipes can be composed to form a pipeline
*/
type Pipe interface {
	Process(in interface{}) (out interface{}, next bool)
}

/*
Pipeline connects multiple pipes in order. The head chan receives incoming items
and tail chan send out items that the pipeline has finished processing.
*/
type Pipeline struct {
	bufferSize int              // Chan buffer size
	head       chan interface{} // Chan representing the head of the pipeline
	tail       chan interface{} // Chan representing the tail of the pipeline
	DebugMode  bool             // Option to log pipeline state transitions, false by default
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
		p.tail, p.DebugMode)
}

/*
AddPipe attaches a pipe to the end of the pipeline.
This will immediately start routing items to this newly attached pipe
*/
func (p *Pipeline) AddPipe(pipe Pipe) *Pipeline {
	oldTail := p.tail
	newTail := make(chan interface{}, p.bufferSize)
	startPipe(oldTail, newTail, pipe)
	p.tail = newTail
	return p
}

/*
AddJunction creates a new Junction to this pipeline and immediately start routing
to that junction
*/
func (p *Pipeline) AddJunction(j Junction) {
	j.route(p.tail)
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
debug Prints log statements if debugLog is true
*/
func (p *Pipeline) debug(values ...string) {
	if p.DebugMode {
		log.Println(values)
	}
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
		startPipe(head, tail, pipe)
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

func startPipe(head, tail chan interface{}, pipe Pipe) {
	go func() {
		for in := range head {
			if out, next := pipe.Process(in); next {
				tail <- out
			}
		}
		close(tail)
	}()
}
