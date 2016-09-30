package gopipe

import "log"

/*
A single pipe component that processes items. Pipes can be composed to form a pipeline
*/
type Pipe interface {
	Process(in chan interface{}) chan interface{}
}

/*
Pipeline connects multiple pipes in order. The head chan receives incoming items
and tail chan send out items that the pipeline has finished processing.
*/
type Pipeline struct {
	head     chan interface{} // Chan representing the head of the pipeline
	tail     chan interface{} // Chan representing the tail of the pipeline
	debugLog bool             // Option to log pipeline state transitions, false by default
}

/*
Enqueue method takes an item one at a time and adds it to the start of the pipeline.
Use AttachSource to attach a chan of incoming items to the pipeline
*/
func (p *Pipeline) EnqueueItem(item interface{}) {
	p.head <- item
}

/*
Close makes sure that the pipeline accepts no further messages.
If the go routine/method writing to the pipeline is still Enqueuing, it will
cause a panic - can't write to a closed channel
*/
func (p *Pipeline) Close() {
	close(p.head)
}

/*
AttachSource accepts the source channel as the entry point to the pipeline
*/
func (p *Pipeline) AttachSource(source chan interface{}) {
	p.debug("Attaching source channel to pipe")
	go func() {
		for item := range source {
			p.head <- item
		}
		p.debug("Pipe source closed. Closing pipeline")
		p.Close()
		return
	}()
}

/*
AttachSink takes a terminating channel and dequeues the messages
from the pipeline onto that channel.
*/
func (p *Pipeline) AttachSink(out chan interface{}) {
	go func() {
		for item := range p.tail {
			out <- item
		}
		p.debug("Shutting down Pipeline Sink")
		close(out)
		return
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
		return
	}()
	return
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
Enable debug logging on this pipeline
*/
func (p *Pipeline) Debug() {
	p.debugLog = true
}

/*
NewPipeline takes multiple pipes in-order and connects them to form a pipeline.
Enqueue and Dequeue methods are used to attach source/sink to the pipeline.
If debugLog is true, logs state transitions to stdout.
*/
func NewPipeline(pipes ...Pipe) Pipeline {
	head := make(chan interface{})
	var connector chan interface{}
	for _, pipe := range pipes {
		if connector == nil {
			connector = pipe.Process(head)
		} else {
			connector = pipe.Process(connector)
		}
	}
	return Pipeline{head: head, tail: connector}
}
