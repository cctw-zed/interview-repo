package main

import (
	"fmt"
	"sync"
)

// Exercise 9: Fan-in, Fan-out Pattern
//
// Problem:
// Implement a pipeline of goroutines that demonstrates the fan-out, fan-in pattern.
// 1. A "producer" goroutine generates a sequence of numbers and sends them to a channel.
// 2. A "squarer" stage (fan-out) consists of multiple goroutines. Each reads numbers from the producer's channel,
//    squares them, and sends the results to its own output channel.
// 3. A "merger" stage (fan-in) reads the squared numbers from all the squarer goroutines' channels
//    and prints them to the console.
//
// Requirements:
// 1. Create a `producer` function that generates integers (e.g., 1 to 10) and sends them to a channel.
// 2. Create a `squarer` function that reads from an input channel, squares the numbers, and sends them to an output channel.
//    Start multiple `squarer` goroutines.
// 3. Create a `merger` function that takes multiple input channels (from the squarers), reads from them,
//    and sends all values to a single output channel.
// 4. The main goroutine should set up the pipeline and print the final results. The order of the printed numbers does not matter.
//
// Concepts to use:
// - Goroutines
// - Channels
// - `sync.WaitGroup`
// - Fan-out (multiple goroutines reading from one channel)
// - Fan-in (one goroutine reading from multiple channels)

func main() {
	// TODO: Implement the fan-in, fan-out pipeline.
	fmt.Println("Fan-in, Fan-out Pipeline")
	producerChan := producer()
	var squarerChans []<-chan int
	for i := 0; i < 3; i++ {
		squarerChans = append(squarerChans, squarer(producerChan))
	}
	mergeChan := merge(squarerChans...)
	for v := range mergeChan {
		fmt.Println(v)
	}
}

func producer() <-chan int {
	ch := make(chan int, 5)
	go func() {
		defer close(ch)
		for i := 1; i <= 1000; i++ {
			ch <- i
		}
	}()
	return ch
}

func squarer(input <-chan int) <-chan int {
	ch := make(chan int, 5)
	go func() {
		defer close(ch)
		for v := range input {
			ch <- v
		}
	}()
	return ch
}

func merge(inputs ...<-chan int) <-chan int {
	ch := make(chan int, 5)
	var wg sync.WaitGroup
	wg.Add(len(inputs))
	for _, input := range inputs {
		input := input
		go func() {
			defer wg.Done()
			for v := range input {
				ch <- v
			}
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}
