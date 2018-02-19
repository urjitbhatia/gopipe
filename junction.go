package gopipe

/*
Junction joins one or more pipelines together
*/
type Junction struct {
	routingFn func(interface{}) interface{}
	router    map[interface{}]*Pipeline
	in        chan interface{}
}

func (j *Junction) AddPipeline(key interface{}, p *Pipeline) {
	j.router[key] = p
}
