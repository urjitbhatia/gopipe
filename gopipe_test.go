package gopipe_test

import (
	"log"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/urjitbhatia/gopipe"
)

var _ = Describe("Pipeline", func() {
	log.SetOutput(GinkgoWriter)

	Describe("without pipes", func() {
		It("is just a channel", func() {
			p := NewPipeline()
			go p.Enqueue("foo")
			Eventually(p.Dequeue).Should(Equal("foo"))
			Expect(p.DequeueTimeout(1 * time.Millisecond)).Should(BeNil())
		})

		It("is just a channel with the right buffer size", func() {
			p := NewBufferedPipeline(2)
			p.Enqueue("foo")
			p.Enqueue("bar")
			Expect(p.Dequeue()).Should(Equal("foo"))
			Expect(p.Dequeue()).Should(Equal("bar"))
			Expect(p.DequeueTimeout(1 * time.Millisecond)).Should(BeNil())
			p.Close()
		})
	})

	Describe("Pipeline with test stub pipes", func() {
		Context("Enque items", func() {
			It("Dequeue", func() {
				max := 20
				dp := doublingPipe{}
				sp := subtractingPipe{}
				pipeline := NewPipeline(dp, sp)
				pipeline.DebugMode = true

				pipeinput := intGenerator(max)
				pipeline.AttachSource(pipeinput)

				pipeout := make(chan interface{})
				pipeline.AttachSink(pipeout)

				for start := 0; start < max; start++ {
					outVal := <-pipeout
					Expect(outVal).To(Equal((start * 2) - 1))
				}
			})

			It("SinkFanOut", func() {
				max := 20
				pp := pluralizingPipe{}
				pipeline := NewPipeline(pp)

				pipeinput := animalGenerator(max)
				pipeline.AttachSource(pipeinput)

				fanout := make(map[string]chan interface{})
				fanout["dogs"] = make(chan interface{})
				fanout["cats"] = make(chan interface{})
				fanout["toads"] = make(chan interface{})
				unroutedChan := make(chan interface{})
				pipeline.AttachSinkFanOut(fanout, unroutedChan, func(item interface{}) (string, error) {
					if itemString, ok := item.(string); ok {
						return itemString, nil
					}
					return "", nil
				})
				for start := 0; start < max; start++ {
					select {
					case val, more := <-fanout["dogs"]:
						if !more {
							fanout["dogs"] = nil
						} else {
							Expect(val).To(Equal("dogs"))
						}
					case val, more := <-fanout["cats"]:
						if !more {
							fanout["cats"] = nil
						} else {
							Expect(val).To(Equal("cats"))
						}
					case val, more := <-fanout["toads"]:
						if !more {
							fanout["toads"] = nil
						} else {
							Expect(val).To(Equal("toads"))
						}
					case val, more := <-unroutedChan:
						if !more {
							unroutedChan = nil
						} else {
							Expect(val).To(Equal("dinosaurs"))
						}
					}
				}
			})

			It("EnqueItems", func() {
				max := 10
				pp := pluralizingPipe{}
				pipeline := NewPipeline(pp)

				pipeinput := animalGenerator(max)
				go func() {
					for animal := range pipeinput {
						// Drain input and enque one by one
						pipeline.Enqueue(animal)
					}
					pipeline.Close()
				}()

				fanout := make(map[string]chan interface{})
				fanout["dogs"] = make(chan interface{})
				fanout["cats"] = make(chan interface{})
				fanout["toads"] = make(chan interface{})
				unroutedChan := make(chan interface{})
				pipeline.AttachSinkFanOut(fanout, unroutedChan, func(item interface{}) (string, error) {
					if itemString, ok := item.(string); ok {
						return itemString, nil
					}
					return "", nil
				})
				for start := 0; start < max; start++ {
					select {
					case val, more := <-fanout["dogs"]:
						if !more {
							fanout["dogs"] = nil
						} else {
							Expect(val).To(Equal("dogs"))
						}
					case val, more := <-fanout["cats"]:
						if !more {
							fanout["cats"] = nil
						} else {
							Expect(val).To(Equal("cats"))
						}
					case val, more := <-fanout["toads"]:
						if !more {
							fanout["toads"] = nil
						} else {
							Expect(val).To(Equal("toads"))
						}
					case val, more := <-unroutedChan:
						if !more {
							unroutedChan = nil
						} else {
							Expect(val).To(Equal("dinosaurs"))
						}
					}
				}
			})

			It("Dequeue with a tap", func() {
				max := 20
				dp := doublingPipe{}

				// A second doubling pipe that we will attach to a tap
				sp := subtractingPipe{}
				pipeline := NewPipeline(dp, sp)
				pipeline.DebugMode = true

				pipeinput := intGenerator(max)
				pipeline.AttachSource(pipeinput)

				tapOut := make(chan interface{})
				pipeline.AttachTap(tapOut)

				pipeout := make(chan interface{})
				pipeline.AttachSink(pipeout)

				for start := 0; start < max; start++ {
					outVal := <-pipeout
					tapVal := <-tapOut
					Expect(outVal).To(Equal(tapVal))
					Expect(outVal).To(Equal((start * 2) - 1))
				}
			})
		})
	})

	Describe("Add Pipe iterface", func() {
		It("works", func() {
			p := NewPipeline()
			// Add two doubling pipes
			p.AddPipe(doublingPipe{}).AddPipe(doublingPipe{})
			p.Enqueue(2)

			Expect(p.Dequeue()).Should(Equal(8))
		})
		It("is easy to add junctions", func() {
			pOne := NewPipeline()

			pOne.AddPipe(subtractingPipe{})
			routingFn := func(val interface{}) interface{} {
				i, _ := val.(int)
				if i%2 == 0 {
					return "even"
				}
				return "odd"
			}
			j := NewJunction(RoutingFunc(routingFn))
			pTwoEven := NewPipeline()
			pTwoEven.AddPipe(subtractingPipe{})

			pTwoOdd := NewPipeline()
			pTwoOdd.AddPipe(doublingPipe{})

			j.AddPipeline("even", pTwoEven).AddPipeline("odd", pTwoOdd)
			pOne.AddJunction(j)

			pOne.Enqueue(3)
			Expect(pTwoEven.Dequeue()).Should(Equal(1))

			pOne.Enqueue(2)
			Expect(pTwoOdd.Dequeue()).Should(Equal(2))
		})

		It("junction keeps works with bad routing fn", func() {
			pOne := NewPipeline()

			pOne.AddPipe(subtractingPipe{})
			routingFn := func(val interface{}) interface{} {
				i, _ := val.(int)
				if i%2 == 0 {
					return "even"
				}
				return "odd"
			}
			j := NewJunction(RoutingFunc(routingFn))
			pTwoEven := NewPipeline()
			pTwoEven.AddPipe(subtractingPipe{})

			pTwoOdd := NewPipeline()
			pTwoOdd.AddPipe(doublingPipe{})

			j.AddPipeline("even", pTwoEven)
			j.AddPipeline("this should've said odd", pTwoOdd)
			pOne.AddJunction(j)

			pOne.Enqueue(3)
			Expect(pTwoEven.Dequeue()).Should(Equal(1))

			pOne.Enqueue(2)
			Eventually(func() interface{} {
				return pTwoOdd.DequeueTimeout(1 * time.Millisecond)
			}).ShouldNot(Equal(2))
		})

		It("works with multiple junctions in the pipeline", func() {
			/*
														|J| > 3	|-> double
								 |J|even	|-> double	|J| <=3	|-> subtract
				in -> subtract 1 |J|
								 |J|odd		|-> subtract
			*/
			p := NewPipeline(subtractingPipe{})

			pTwoEven := NewPipeline(doublingPipe{})
			pTwoOdd := NewPipeline(subtractingPipe{})

			pThreeGt := NewPipeline(doublingPipe{})
			pThreeLe := NewPipeline(subtractingPipe{})

			evenOddFn := RoutingFunc(func(val interface{}) interface{} {
				if i, _ := val.(int); i%2 == 0 {
					return "even"
				}
				return "odd"
			})

			jOne := NewJunction(evenOddFn)
			jOne.AddPipeline("even", pTwoEven).AddPipeline("odd", pTwoOdd)
			p.AddJunction(jOne)

			greaterThan3Fn := RoutingFunc(func(val interface{}) interface{} {
				if i, _ := val.(int); i > 3 {
					return true
				}
				return false
			})

			jTwo := NewJunction(greaterThan3Fn)
			jTwo.AddPipeline(true, pThreeGt).AddPipeline(false, pThreeLe)
			pTwoEven.AddJunction(jTwo)

			p.Enqueue(3)
			Expect(pThreeGt.Dequeue()).Should(Equal(8))
			p.Enqueue(1)
			Expect(pThreeLe.Dequeue()).Should(Equal(-1))

			p.Enqueue(2)
			Expect(pTwoOdd.Dequeue()).Should(Equal(0))
		})
	})

	Describe("Pipeline benchmarks", func() {
		Context("benchmark - 500 samples of 10000 inputs", func() {
			Measure("send messages through a buffered pipeline", func(b Benchmarker) {
				runtime := b.Time("runtime", func() {
					max := 10000
					dp := doublingPipe{}
					sp := subtractingPipe{}
					pipeline := NewBufferedPipeline(200, dp, sp)

					pipeinput := intGenerator(max)
					pipeline.AttachSource(pipeinput)

					pipeout := make(chan interface{})
					pipeline.AttachSink(pipeout)

					for start := 0; start < max; start++ {
						Expect(<-pipeout).To(Equal((start * 2) - 1))
					}
				})

				Ω(runtime.Seconds()).Should(BeNumerically("<", 0.5), "Shouldn't take more than 0.5 sec for 10000 items")
			}, 500)
		})
	})
})
