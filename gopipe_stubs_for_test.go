package gopipe_test

import (
	"log"
)

type doublingPipe struct{}

func (dp doublingPipe) Process(in interface{}) (interface{}, bool) {
	if intval, ok := in.(int); ok {
		return intval * 2, true
	}
	log.Println("not ok")
	return nil, false
}

type subtractingPipe struct{}

func (sp subtractingPipe) Process(in interface{}) (interface{}, bool) {
	if intval, ok := in.(int); ok {
		return intval - 1, true
	}
	log.Println("not ok")
	return nil, false
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

func (pp pluralizingPipe) Process(in interface{}) (interface{}, bool) {
	if strVal, ok := in.(string); ok {
		return strVal + "s", true
	}
	log.Println("non-string input")
	return nil, false
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
