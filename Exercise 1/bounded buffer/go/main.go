package main

import (
	"fmt"
	"time"
)

func producer(boundedBuf chan int) {

	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("[producer]: pushing %d\n", i)
		boundedBuf <- i
	}

}

func consumer(boundedBuf chan int) {

	time.Sleep(1 * time.Second)
	for {
		i := <-boundedBuf //TODO: get real value from buffer
		fmt.Printf("[consumer]: %d\n", i)
		time.Sleep(50 * time.Millisecond)
	}

}

func main() {

	// TODO: make a bounded buffer
	var boundedBuf chan int = make(chan int, 5)

	go consumer(boundedBuf)
	go producer(boundedBuf)

	select {}
}
