package gopipe

import (
	"fmt"
	"log"
)

/*
Junction joins one or more pipelines together
*/
type Junction struct {
	routingFn func(interface{}) interface{}
	router    map[interface{}]*Pipeline
	DebugMode bool // Option to log pipeline state transitions, false by default
}

/*
RoutingFunc takes in an item flowing through the pipeline and maps it to another item.
The output item is used to then route data flowing through the Junction to the right
pipeline attached to it
*/
type RoutingFunc func(in interface{}) interface{}

/*
NewJunction creates a new Junction
*/
func NewJunction(rFunc RoutingFunc) Junction {
	return Junction{routingFn: rFunc, router: make(map[interface{}]*Pipeline)}
}

/*
AddPipeline adds a pipeline to a junction. Items that output the given key
when fed into the routing function for this junction are routed the given pipeline
*/
func (j *Junction) AddPipeline(key interface{}, p *Pipeline) *Junction {
	j.router[key] = p
	return j
}

// route will attach to a pipeline and start routing messages
// to other pipelines based on routing function
func (j *Junction) route(in chan interface{}) {
	go func() {
		for item := range in {
			routingKey := j.routingFn(item)
			j.debug(fmt.Sprintf("routing key: %v", routingKey))
			if dest, ok := j.router[routingKey]; ok {
				dest.Enqueue(item)
			} else {
				j.debug(fmt.Sprintf("Junction discarding item: %+v no dest pipeline for routing key: %+v",
					item, routingKey))
			}
		}
	}()
}

/*
debug Prints log statements if debugLog is true
*/
func (j *Junction) debug(values ...string) {
	if j.DebugMode {
		log.Println(values)
	}
}
