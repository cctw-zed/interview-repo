package main

import (
	"fmt"
	"sync"
)

func main() {

	counter := NewCounter()

	go func() {
		for i := 0; i < 1000; i++ {
			counter.Incr()
			fmt.Println(counter.Get())
		}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			counter.Decr()
			fmt.Println(counter.Get())
		}
	}()
}

type Counter struct {
	val int
	mu  sync.RWMutex
}

func NewCounter() *Counter {
	return &Counter{}
}

func (c *Counter) Incr() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.val++
}

func (c *Counter) IncrBy(val int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.val += val
}

func (c *Counter) Decr() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.val--
}

func (c *Counter) DecrBy(val int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.val -= val
}

func (c *Counter) Get() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.val
}
