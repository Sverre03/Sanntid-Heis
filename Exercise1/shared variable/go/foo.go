// Use `go run foo.go` to run your program

package main

import (
	. "fmt"
	"runtime"
	"time"
)

func incrementing(channel chan int, finish chan int) {
	for j := 0; j < 1000000; j++ {
		channel <- 1
	}
	finish <- 1
}

func decrementing(channel chan int, finish chan int) {
	for j := 0; j < 1000000; j++ {
		channel <- -1
	}
	finish <- 1
}

func server(channel chan int, finish chan int, iChan chan int) {
	var i int = 0
	var finishCounter int = 0
	for {
		select {
		case tempChannel := <-channel:
			if tempChannel == 1 {
				i++
			}
			if tempChannel == -1 {
				i--
			}
		case <-finish:
			finishCounter++
		case <-finish:
			finishCounter++
		}
		if finishCounter == 2 {
			iChan <- i
			return
		}
	}
}

func main() {
	// What does GOMAXPROCS do? What happens if you set it to 1?
	// GOMAXPROCS sets the maximum number of processes which can be run simultaneously.
	// Setting it to 1 means the tasks are performed sequentially, as we only have one process.
	runtime.GOMAXPROCS(3)

	var channel = make(chan int)
	var finish = make(chan int)
	var iChan = make(chan int)

	// TODO: Spawn both functions as goroutines
	go incrementing(channel, finish)
	go decrementing(channel, finish)
	go server(channel, finish, iChan)

	// We have no direct way to wait for the completion of a goroutine (without additional synchronization of some sort)
	// We will do it properly with channels soon. For now: Sleep.
	Println("The magic number is:", <-iChan)
	time.Sleep(500 * time.Millisecond)
}
