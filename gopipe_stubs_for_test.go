package gopipe_test

import (
	"log"
)

type doublingPipe struct{}

func (dp doublingPipe) Process(in chan interface{}, out chan interface{}) {
	for item := range in {
		if intval, ok := item.(int); ok {
			out <- intval * 2
		} else {
			log.Println("not ok")
		}
	}
}

type subtractingPipe struct{}

func (sp subtractingPipe) Process(in chan interface{}, out chan interface{}) {
	for item := range in {
		if intval, ok := item.(int); ok {
			out <- intval - 1
		} else {
			log.Println("not ok")
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
	for item := range in {
		if strVal, ok := item.(string); ok {
			out <- strVal + "s"
		} else {
			log.Println("non-string input")
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
