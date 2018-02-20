package gopipe

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
