# gopipe
A stream-filter like pipeline primitive for go

[![Build Status](https://travis-ci.org/urjitbhatia/gopipe.svg?branch=master)](https://travis-ci.org/urjitbhatia/gopipe)
[![GoDoc](https://godoc.org/github.com/urjitbhatia/gopipe?status.svg)](https://godoc.org/github.com/urjitbhatia/gopipe)

Gopipe exposes a simple interface that your "Pipe" must implement:
```go
/*
A single pipe component that processes items. Pipes can be composed to form a pipeline
*/
type Pipe interface {
	Process(in chan interface{}, out chan interface{})
}
```

Any such pipe can then be combined into a pipeline like:
```go
// Make a pipeline
pipeline := gopipe.NewPipeline(
  jsonUnmarshalPipe,
  redisWriterPipe,
  logWriterPipe
)

// Or Make a Buffered Pipeline
// This allows up to bufSize elements to queue at *Each Pipe stage
bufSize := 10
// Buffersize 10 throughout the pipe
bufP := gopipe.NewBufferedPipeline(bufSize, redisWriterPipe, logWriterPipe)

// Attach some source
jobs := make(chan interface{})
pipeline.AttachSource(jobs)

// Attach Sink
processedJobs := make(chan interface{})
pipeline.AttachSink(processedJobs)

// Or Enqueue from somewhere (Block if the pipeline has no capacity)
pipeline.Enqueue("foo")

// And Dequeue (Blocks if nothing is flowing)
bar := pipeline.Dequeue()

// Dequeue with timeout
baz := pipeline.DequeueTimeout(10 * time.Millisecond)
```

# Complex pipelining

You can also create a "routing" junction and attach other Pipelines downstream.
```go

// Create a RoutingFunc func(interface{}) interface{}
routingFn := RoutingFunc(func(val interface{}) interface{} {
  if val > 10 && val < 100 {
    return "smallishNumber"
  } else if val >= 100 {
    return "biggishNumber"
  }
  return "eh!", errors.New("dwarfnumber")
})

// Create a junction
j := NewJunction(routingFn)
j.AddPipeline("smallishNumber", NewPipeline(smallNumPipe)).AddPipeline("biggishNumber", NewPipeline(bigNumPipe)

// Now attach the junction - as soon as this is attached, data will start flowing
pipeline.AddJunction(j)
```

# Example Pipe:

This is a pipe that takes in integers and doubles them. If the input is invalid, it effectively "filters" it from going down the pipeline. In a more complex scenario, you can update the incoming structs with error flags etc and might still want to propagate it dowstream.

To filter, simply don't put it on the `out` chan.

```go
type doublingPipe struct{}

func (dp doublingPipe) Process(in chan interface{}, out chan interface{}) {
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
			  log.Println("not ok - filtering...")
			}
		}
	}
}
```

# More Examples:

- [See more examples](./gopipe_example_test.go)
- [Various Pipe examples](./gopipe_stubs_for_test.go)
- [More examples in tests](./gopipe_test.go)
