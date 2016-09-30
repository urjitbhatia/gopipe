package gopipe_test

import (
	. "github.com/urjitbhatia/gopipe"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"log"
)

type doublingPipe struct{}

func (dp doublingPipe) Process(in chan interface{}) chan interface{} {
	out := make(chan interface{})
	go func() {
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
	}()
	return out
}

type subtractingPipe struct{}

func (sp subtractingPipe) Process(in chan interface{}) chan interface{} {
	out := make(chan interface{})
	go func() {
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
	}()
	return out
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

func (pp pluralizingPipe) Process(in chan interface{}) chan interface{} {
	out := make(chan interface{})
	go func() {
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
	}()
	return out
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

				dp := doublingPipe{}
				sp := subtractingPipe{}
				pipeline := NewPipeline(dp, sp)

				pipeinput := intGenerator(20)
				pipeline.AttachSource(pipeinput)

				pipeout := make(chan interface{})
				pipeline.AttachSink(pipeout)

				var start = 0
			outloop:
				for {
					select {
					case val, more := <-pipeout:
						if !more {
							pipeout = nil
							break outloop
						}
						Expect(val).To(Equal((start * 2) - 1))
						start++
					}
				}
			})

			It("SinkFanOut", func() {

				pp := pluralizingPipe{}
				pipeline := NewPipeline(pp)

				pipeinput := animalGenerator(20)
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
				var chanCloseCount = 0
			outloop:
				for {
					select {
					case val, more := <-fanout["dogs"]:
						if !more {
							fanout["dogs"] = nil
							chanCloseCount++
						} else {
							Expect(val).To(Equal("dogs"))
						}
					case val, more := <-fanout["cats"]:
						if !more {
							fanout["cats"] = nil
							chanCloseCount++
						} else {
							Expect(val).To(Equal("cats"))
						}
					case val, more := <-fanout["toads"]:
						if !more {
							fanout["toads"] = nil
							chanCloseCount++
						} else {
							Expect(val).To(Equal("toads"))
						}
					case val, more := <-unroutedChan:
						if !more {
							unroutedChan = nil
							chanCloseCount++
						} else {
							Expect(val).To(Equal("dinosaurs"))
						}
					default:
						if chanCloseCount == 4 {
							break outloop
						}
					}
				}
			})

			It("EnqueItems", func() {

				pp := pluralizingPipe{}
				pipeline := NewPipeline(pp)

				pipeinput := animalGenerator(10)
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
				var chanCloseCount = 0
			outloop:
				for {
					select {
					case val, more := <-fanout["dogs"]:
						if !more {
							fanout["dogs"] = nil
							chanCloseCount++
						} else {
							Expect(val).To(Equal("dogs"))
						}
					case val, more := <-fanout["cats"]:
						if !more {
							fanout["cats"] = nil
							chanCloseCount++
						} else {
							Expect(val).To(Equal("cats"))
						}
					case val, more := <-fanout["toads"]:
						if !more {
							fanout["toads"] = nil
							chanCloseCount++
						} else {
							Expect(val).To(Equal("toads"))
						}
					case val, more := <-unroutedChan:
						if !more {
							unroutedChan = nil
							chanCloseCount++
						} else {
							Expect(val).To(Equal("dinosaurs"))
						}
					default:
						if chanCloseCount == 4 {
							break outloop
						}
					}
				}
			})
		})
	})

	Describe("Pipeline benchmarks", func() {
		Context("benchmark - 500 samples of 10000 inputs", func() {
			Measure("send messages through the pipeline", func(b Benchmarker) {
				runtime := b.Time("runtime", func() {
					dp := doublingPipe{}
					sp := subtractingPipe{}
					pipeline := NewPipeline(dp, sp)

					pipeinput := intGenerator(10000)
					pipeline.AttachSource(pipeinput)

					pipeout := make(chan interface{})
					pipeline.AttachSink(pipeout)

					var start = 0
				outloop:
					for {
						select {
						case val, more := <-pipeout:
							if !more {
								pipeout = nil
								break outloop
							}
							Expect(val).To(Equal((start * 2) - 1))
							start++
						}
					}
				})

				Ω(runtime.Seconds()).Should(BeNumerically("<", 0.03), "Shouldn't take more than 0.03 sec.")
			}, 500)
		})
	})
})
