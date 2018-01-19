# gopipe
A stream-filter like pipeline primitive for go

Godoc documentation: [![GoDoc](https://godoc.org/github.com/urjitbhatia/gopipe?status.svg)](https://godoc.org/github.com/urjitbhatia/gopipe)

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

// Attach some source
jobs := make(chan interface{})
pipeline.AttachSource(jobs)

// Attach Sink
processedJobs := make(chan interface{})
pipeline.AttachSink(processedJobs)
```

# Complex pipelining

You can also create a "routing" junction using pipes by using `AttachSinkFanOut` instead of a simple `AttachSink`.
The first argument for this is a `map` of `channels` like:
```go
chanfan := make(map[string]chan interface{})
chanfan["smallishNumber"] = make(chan interface{})
chanfan["biggishNumber"] = make(chan interface{})

// Create an error/unrouted msg channel
unroutedChan := make(chan interface{})

// Create a routingFunction satisfying the interface: routingFunc func(interface{}) (string, error)
routingFn := func(val interface{}) (string, error) {
  if val > 10 && val < 100 {
    return "smallishNumber", nil
  } else if val >= 100 {
    return "biggishNumber", nil
  }
  return "eh!", errors.New("dwarfnumber")
}

// Now call the AttachSinkFanOut
pipeline.AttachSinkFanOut(chanfan, unroutedChan, routingFn)
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
      // This has the effect of "filtering" the input
      // because its not passed down the pipeline anymore.
			  log.Println("not ok")
			}
		}
	}
}
```
