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
