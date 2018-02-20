package gopipe

/*
Junction joins one or more pipelines together
*/
type Junction struct {
	routingFn func(interface{}) interface{}
	router    map[interface{}]*Pipeline
	in        chan interface{}
}

/*
AddPipeline adds a pipeline to a junction. Items that output the given key
when fed into the routing function for this junction are routed the given pipeline
*/
func (j *Junction) AddPipeline(key interface{}, p *Pipeline) *Junction {
	j.router[key] = p
	return j
}

/*
RoutingFunc takes in an item flowing through the pipeline and maps it to another item.
The output item is used to then route data flowing through the Junction to the right
pipeline attached to it
*/
type RoutingFunc func(in interface{}) interface{}
