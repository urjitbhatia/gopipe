package gopipe_test

import (
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/urjitbhatia/gopipe"
)

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
				log.Println("not ok")
			}
		}
	}
}

type subtractingPipe struct{}

func (sp subtractingPipe) Process(in chan interface{}, out chan interface{}) {
	for {
		select {
		case item, more := <-in:
			if !more {
				log.Println("Pipe-in closed")
				close(out)
				return
			}
			if intval, ok := item.(int); ok {
				out <- intval - 1
			} else {
				log.Println("not ok")
			}
		}
	}
}

func intGenerator(limit int) (out chan interface{}) {
	out = make(chan interface{})
	go func() {
		defer close(out)
		for i := 0; i < limit; i++ {
			out <- i
		}
	}()
	return
}

type pluralizingPipe struct{}

func (pp pluralizingPipe) Process(in chan interface{}, out chan interface{}) {
	for {
		select {
		case item, more := <-in:
			if !more {
				log.Println("Pipe-in closed")
				close(out)
				return
			}
			if strVal, ok := item.(string); ok {
				out <- strVal + "s"
			} else {
				log.Println("unknown")
			}
		}
	}
}

func animalGenerator(limit int) (out chan interface{}) {
	out = make(chan interface{})
	animals := []string{"cat", "dog", "toad", "dinosaur"}
	go func() {
		defer close(out)
		for i := 0; i < limit; i++ {
			out <- animals[i%len(animals)]
		}
	}()
	return
}

var _ = Describe("Pipeline", func() {
	log.SetOutput(GinkgoWriter)
	Describe("Pipeline with a single pipe", func() {
		Context("Enque items", func() {
			It("Dequeue", func() {
				max := 20
				dp := doublingPipe{}
				sp := subtractingPipe{}
				pipeline := NewPipeline(dp, sp)
				pipeline.Debug()

				pipeinput := intGenerator(max)
				pipeline.AttachSource(pipeinput)

				pipeout := make(chan interface{})
				pipeline.AttachSink(pipeout)

				for start := 0; start < max; start += 1 {
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
				for start := 0; start < max; start += 1 {
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
						pipeline.EnqueueItem(animal)
					}
					pipeline.Close()
					return
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
				for start := 0; start < max; start += 1 {
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
				pipeline.Debug()

				pipeinput := intGenerator(max)
				pipeline.AttachSource(pipeinput)

				tapOut := make(chan interface{})
				pipeline.AttachTap(tapOut)

				pipeout := make(chan interface{})
				pipeline.AttachSink(pipeout)

				for start := 0; start < max; start += 1 {
					outVal := <-pipeout
					tapVal := <-tapOut
					Expect(outVal).To(Equal(tapVal))
					Expect(outVal).To(Equal((start * 2) - 1))
				}
			})
		})
	})

	Describe("Pipeline benchmarks", func() {
		Context("benchmark - 500 samples of 10000 inputs", func() {
			Measure("send messages through a buffered pipeline", func(b Benchmarker) {
				runtime := b.Time("runtime", func() {
					max := 10000
					dp := doublingPipe{}
					sp := subtractingPipe{}
					pipeline := NewBufferedPipe(200, dp, sp)

					pipeinput := intGenerator(max)
					pipeline.AttachSource(pipeinput)

					pipeout := make(chan interface{})
					pipeline.AttachSink(pipeout)

					for start := 0; start < max; start += 1 {
						select {
						case val, more := <-pipeout:
							if !more {
								pipeout = nil
							}
							Expect(val).To(Equal((start * 2) - 1))
						}
					}
				})

				Ω(runtime.Seconds()).Should(BeNumerically("<", 0.5), "Shouldn't take more than 0.5 sec for 10000 ops")
			}, 500)
		})
	})
})
