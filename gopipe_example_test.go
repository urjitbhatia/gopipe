package gopipe_test

import (
	"fmt"
	"time"

	. "github.com/urjitbhatia/gopipe"
)

func ExamplePipeline() {
	max := 4
	dp := doublingPipe{}
	sp := subtractingPipe{}
	pipeline := NewPipeline(dp, sp)

	pipeinput := intGenerator(max)
	pipeline.AttachSource(pipeinput)

	for i := 0; i < max; i++ {
		fmt.Printf("value is: %d\n", pipeline.Dequeue())
	}
	// Output:
	// value is: -1
	// value is: 1
	// value is: 3
	// value is: 5
}

// ExampleEnqueue shows an alternative usage of Pipeline without channels directly.
// Calling Enqueue will put an item in the pipeline and Dequeue will consume it. Both are blocking
// operations.
func ExampleEnqueue() {
	max := 4
	dp := doublingPipe{}
	sp := subtractingPipe{}
	pipeline := NewBufferedPipeline(max, dp, sp)

	for i := range intGenerator(max) {
		pipeline.Enqueue(i)
	}

	for i := 0; i < max-1; i++ {
		fmt.Printf("value is: %d\n", pipeline.Dequeue())
	}
	fmt.Printf("Dequeue valid with timeout: %v\n", pipeline.DequeueTimeout(1*time.Millisecond))
	fmt.Printf("Dequeue with timeout: %v\n", pipeline.DequeueTimeout(1*time.Millisecond))

	// Output:
	// value is: -1
	// value is: 1
	// value is: 3
	// Dequeue valid with timeout: 5
	// Dequeue with timeout: <nil>
}

// ExampleJunction shows how to create a junction and route data across
// the junction to other pipelines
func ExampleJunction() {
	max := 4
	p := NewBufferedPipeline(max)
	rf := RoutingFunc(func(in interface{}) interface{} {
		if i, _ := in.(int); i > 2 {
			return "big"
		}
		return "small"
	})
	j := NewJunction(rf)

	dp := NewPipeline(doublingPipe{})
	sp := NewPipeline(subtractingPipe{})

	// If input is "small" send to doublingPipeline
	// If input is "big" send to subtractingPipeline
	j.AddPipeline("small", dp).AddPipeline("big", sp)
	p.AddJunction(j)

	for i := range intGenerator(max) {
		p.Enqueue(i)
	}

	fmt.Println("Small pipeline got: ", dp.Dequeue())
	fmt.Println("Small pipeline got: ", dp.Dequeue())
	fmt.Println("Small pipeline got: ", dp.Dequeue())
	fmt.Println("Big pipeline got: ", sp.Dequeue())
	// Output:
	// Small pipeline got:  0
	// Small pipeline got:  2
	// Small pipeline got:  4
	// Big pipeline got:  2
}
