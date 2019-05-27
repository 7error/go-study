<!--
 Copyright 2019 Yandy Ramirez
 
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
 
     http://www.apache.org/licenses/LICENSE-2.0
 
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->

## Go's Concurrency Building Blocks

### Goroutines

A `goroutine` is a function that is running concurrently (not necessarily in parallel!) alongside other functions (code). A `goroutine` is started by simply placing the `go` keyword before a function.

```go
package main

import (
    "fmt"
    "sync"
)

func main() {
    go sayHello()
}

func sayHello() {
    fmt.Println("hello")
}
```

Anonymous functions can also be used to spawn `goroutines`.

```go
package main

import (
    "fmt"
    "sync"
)

func main() {
    go func() {
        fmt.Println("hello")
    }() // parentheses are necessary to create self-invoking anonymous functions
}
```

The above examples have a problem, it's undetermined if the `sayHello` portion will execute at all before the `main` goroutine exits. Once the `main goroutine` exits, all other `goroutines` scheduled or not will also terminate. The outcome of the example is unpredictable.

`Goroutines` are not `OS Threads`, and they're not exactly green threads (threads that are managed by a language's runtime) they're a higher level of abstraction known as `coroutines`. They're small `subroutines` (functions, closures, methods) that are non-preemptive, means they cannot be interrupted. They do however have multiple points which allow for suspension or reentry.

Go's runtime automatically observes the runtime behaviour of `goroutines` and automatically suspends them when they block and then resume when they become unblocked.

`Go's` mechanism for hosting goroutines is an implementation of what's called an M:N scheduler, which means it maps `M` green threads to `N` OS threads. Go follows a `fork-join` model of concurrency. This means that at any future point in the program it can split into separate branches. `Join` means that a some point, these branches come back together. Visual representation below. The `Go` keyword is what creates a fork, and the forked threads of execution are `goroutines`.

![fork join representation](/images/fork-join-goroutines.png)

In order to create a `join point` for the goroutines and the above `sayHello` example, the `goroutines` need to be `synchronized`. There are a few ways to do this, let's use the `sync` package for now. Here's a fixed version of the example.

```go
package main

import (
    "fmt"
    "sync"
)

func main() {
    var wg sync.WaitGroup

    wg.Add(1)
    go func() {
        defer wg.Done()
        fmt.Println("hello")
    }()
    wg.Wait() // join point
}
```

Understanding closures and `goroutines`.

```go
package main

import (
    "fmt"
    "sync"
)

func main() {
    var wg sync.WaitGroup

    for _, salutation := range []string{"hello", "greetings", "good day"} {
    wg.Add(1)
    go func() {
        defer wg.Done()
        fmt.Println(salutation)
    }()
    }
    wg.Wait() // join point
}
```

When this program is run, the output is unexpectedly this...

```shell
good day
good day
good day
```

This is probably due to the loop exiting before the `Goroutines` have a chance to execute. Go is pretty observant and notices that `salutation` is still in use, moves this address to the heap to avoid a panic. Though the last member of the array is the one output three times.

The `idiomatic` way of writing this loop is to make a copy of salutation and pass that into the closure as an argument.

```go
package main

import (
    "fmt"
    "sync"
)

func main() {
    var wg sync.WaitGroup

    for _, salutation := range []string{"hello", "greetings", "good day"} {
    wg.Add(1)
    go func(salutation string) {
        defer wg.Done()
        fmt.Println(salutation)
    }()
    }
    wg.Wait(salutation) // join point
}
```

`Goroutines` are not garbage collected like other `go` types. Goroutines are also small, which means many of them can be run without fear of running out of memory. Lets find out the size of given goroutines.

```go
package main

import (
	"fmt"
	"runtime"
	"sync"
)

func main() {
	memConsumed := func() uint64 {
		runtime.GC()
		var s runtime.MemStats
		runtime.ReadMemStats(&s)
		return s.Sys
	}

	var c <-chan interface{}
	var wg sync.WaitGroup
	noop := func() { wg.Done(); <-c } // goroutine that will never exit

	const numGoroutines = 1e4 // define number of goroutines to create
	wg.Add(numGoroutines)
	before := memConsumed() // measure memory consumed before creating goroutines
	for i := numGoroutines; i > 0; i-- {
		go noop()
	}
	wg.Wait()
	after := memConsumed() // measure memory consumed after creating goroutines
	fmt.Printf("%.3fkb", float64(after-before)/numGoroutines/1000)
}
```

## The sync Package

The `sync` package contains primitives that are most useful for low-level memory access synchronization.

### WaitGroup

The `WaitGroup` is a great way to wait for a set of concurrent operations to complete when you either don't care about the result of the concurrent operation, or you have other means of collecting their result. Here's a small example of using `WaitGroup` to wait for `goroutines` to complete.

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup

	wg.Add(1) // add one goroutine to the counter
	go func() {
		defer wg.Done() // call Done using defer, when goroutine finishes, counter is decremented by 1
		fmt.Println("1st goroutine sleeping...")
		time.Sleep(1)
	}()

	wg.Add(1) // add one goroutine to the counter
	go func() {
		defer wg.Done() // call Done using defer, when goroutine finishes, counter is decremented by 1
		fmt.Println("2nd goroutine sleeping...")
		time.Sleep(2)
	}()

	wg.Wait() // wait blocks the main goroutine until all other goroutines complete
	fmt.Println("All goroutines complete.")
}
```

It's important that the calls to `Add` are done outside the `goroutines`, otherwise a race condition can be introduced. This is because there's no guarantee in the order of execution for these child goroutines.

### Mutex

A `Mutex` stands for `mutual exclusion` and it's a way to guard critical sections of a program. A critical part of a program is a piece of code that requires exclusive access to a *shared* resource. Sample of a program using `Mutex` to synchronize access to a a variable being incremented and decremented.

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var count int
	var lock sync.Mutex

	increment := func() {
		lock.Lock()             // request exclusive use of the critical section, the count variable
		defer lock.Unlock()     // indicate done when access to the variable is complete
		count++
		fmt.Printf("Incrementing: %d\n", count)
	}

	decrement := func() {
		lock.Lock()             // request exclusive use of the critical section, the count variable
		defer lock.Unlock()     // indicate done when access to the variable is complete
		count--
		fmt.Printf("Decrementing: %d\n", count)
	}

	// Increment
	var arithmetic sync.WaitGroup
	for i := 0; i <= 5; i++ {
		arithmetic.Add(1)
		go func() {
			defer arithmetic.Done()
			increment()
		}()
	}

	// Decrement
	for i := 0; i <= 5; i++ {
		arithmetic.Add(1)
		go func() {
			defer arithmetic.Done()
			decrement()
		}()
	}

	arithmetic.Wait()
	fmt.Println("Arithmetic complete.")
}
```

Critical sections are so named because they reflect a bottleneck in your program. It is somewhat expensive to enter and exit a critical section, and so generally people attempt to minimize the time spent in critical sections.

### RWMutex

The sync.RWMutex is conceptually the same thing as a Mutex: it guards access to memory; however, RWMutex gives you a little bit more control over the memory. You can request a lock for reading, in which case you will be granted access unless the lock is being held for writing. Here’s an example that demonstrates a producer that is less active than the numerous consumers the code creates.

```go
package main

import (
	"fmt"
	"math"
	"os"
	"sync"
	"text/tabwriter"
	"time"
)

func main() {
	producer := func(wg *sync.WaitGroup, l sync.Locker) { // 1
		defer wg.Done()
		for i := 5; i > 0; i-- {
			l.Lock()
			l.Unlock()
			time.Sleep(1) // 2
		}
	}

	observer := func(wg *sync.WaitGroup, l sync.Locker) {
		defer wg.Done()
		l.Lock()
		defer l.Unlock()
	}

	test := func(count int, mutex, rwMutex sync.Locker) time.Duration {
		var wg sync.WaitGroup
		wg.Add(count + 1)
		beginTestTime := time.Now()
		go producer(&wg, mutex)
		for i := count; i > 0; i-- {
			go observer(&wg, rwMutex)
		}

		wg.Wait()
		return time.Since(beginTestTime)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)
	defer tw.Flush()

	var m sync.RWMutex
	fmt.Fprintf(tw, "Readers\tRWMutex\tMutex\n")
	for i := 0; i < 20; i++ {
		count := int(math.Pow(2, float64(i)))
		fmt.Fprintf(
			tw,
			"%d\t%v\t%v\n",
			count,
			test(count, &m, m.RLocker()),
			test(count, &m, &m),
		)
	}
}
// 1: The producer function's second parameter is of the type sync.Locker.
//    This interface has two methods, Lock and Unlock, which the Mutex and RWMutex types satisfy.
//
// 2: Make the producer sleep for one second to make it less active than the observer goroutines.
```

### Cond

Is a rendezvous point for goroutines waiting for or announcing the occurrence of an event. An `event` is any arbitrary signal between two or more goroutines that carries no information other than the fact that it has occurred. The `Cond` type allows a `goroutine` to sleep and wake up when it has things to do.

Bellow's an example of a `goroutine` that is waiting for a signal and a `goroutine` that is sending a signal.

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	c := sync.NewCond(&sync.Mutex{}) // 1: create condition using standard sync.Mutex
	queue := make([]interface{}, 0, 10) // 2: create a slice with a length of zero and a capacity of 10

	removeFromQueue := func(delay time.Duration) {
		time.Sleep(delay)
		c.L.Lock() // 8: enter the critical section for the condition again to modify data
		queue = queue[1:] // 9: simulate dequeuing an item by reassigning the head of the slice to the second item
		fmt.Println("Removed from queue")
		c.L.Unlock() // 10: exit the condition's critical section
		c.Signal() // 11: let a goroutine in waiting know that something has occurred
	}

	for i := 0; i < 10; i++ {
		c.L.Lock() // 3: enter the critical section for the condition by calling Lock
		for len(queue) == 2 { // 4: check the length of the queue in the loop
			c.Wait() // 5: call Wait which will suspend the main goroutine until a signal on condition is seen
		}
		fmt.Println("Adding to queue")
		queue = append(queue, struct{}{})
		go removeFromQueue(1 * time.Second) // 6: create a new goroutine that will dequeue an element after one second
		c.L.Unlock() // 7: exit the condition's critical section
	}
}
```

## Once

`sync.Once` is a type that utilizes some `sync` primitives internally to ensure that only one call to `Do` ever calls the function passed in.

Example of a a function being called only once.

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	var count int

	increment := func() {
		count++
	}

	var once sync.Once

	var increments sync.WaitGroup
	increments.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer increments.Done()
			once.Do(increment) // sync.Once guarrantees that your function is only called once
		}()
	}

	increments.Wait()
	fmt.Printf("Count is %d\n", count)
}

// Output: 1
```

## Channels

Go `channels` are one of the primitives for `goroutine` synchronization. They can be used to synchronize access of memory, but are best suited to communicated information between goroutines. `Channels` serve as a conduit of information; values may be passed along the channel, and then read out downstream. `Channels` can be omnidirectional or unidirectional, meaning a channel can both receive and send data, or a channel can receive or send data. `Channels` are typed like any other `Go` type, can also hold the `interface{}` which means any type. Here's a simple channel example.

```go
package main

import (
    "fmt"
)

func main() {
    stringStream := make(chan string) // declare an unbuffered channel of string
    go func() {
        stringStream <- "Hello channels!" // pass string data into the channel
    }()

    fmt.Println(<-stringStream) // read from the channel
}
```

`Channels` are meant to be blocking, meaning that any attempt to read from channel will wait until there's data to be read, and any attempt to write to a full channel will wait until the channel's been freed. This is why the previous example works without any `sync.Add` or `sync.Wait` or `sync.Mutex Lock`.

The `receive from <-` can also be used to receive two values, the data from the `channel` and the `ok` value to indicate success.

```go
package main

import (
	"fmt"
)

func main() {
	stringStream := make(chan string)
	go func() {
		stringStream <- "Hello channels!"
	}()
	salutation, ok := <-stringStream
	fmt.Printf("(%v): %v", ok, salutation)
}
// Output: (true): Hello channels!
```

The second return value is a way for a read operation to indicate whether the read off the channel was a value generated by a write elsewhere in the process, or a default value generated from a closed channel. One can read from a `closed channel` as well.

```go
package main

import (
	"fmt"
)

func main() {
	intStream := make(chan int)
	close(intStream)
	integer, ok := <-intStream // read from a closed stream
	fmt.Printf("(%v): %v", ok, integer)
}
// Output: (false): 0
```

One can `range over channels` as well, stopping once the channel is closed.

```go
package main

import (
	"fmt"
)

func main() {
	intStream := make(chan int)
	go func() {
		defer close(intStream) // ensure that the channel is closed before we exit the goroutine
		for i := 1; i <= 5; i++ {
			intStream <- i
		}
	}()

	for integer := range intStream { // range over stream (channel)
		fmt.Printf("%v ", integer)
	}
}
// Output: 1 2 3 4 5
```

Multiple `goroutines` can be unblocked at once by closing a `channel` that they depend on. A closed channel can be read from an unlimited amount of times, it's easier, cheaper and faster than writing N times.

```go
package main

import (
	"fmt"
	"sync"
)

func main() {
	begin := make(chan interface{})
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-begin // the goroutine waits until it is told it can continue
			fmt.Printf("%v has begun\n", i)
		}(i)
	}

	fmt.Println("Unblocking goroutines...")
	close(begin) // close the channel, which unblocks all the goroutines at once
	wg.Wait()
}
```

We can also create buffered channels, which are channels that are given a capacity when they’re instantiated. This means that even if no reads are performed on the channel, a goroutine can still perform n writes, where n is the capacity of the buffered channel. Here’s how to declare and instantiate one:

```go
dataStream := make(chan interface{}, 4) // create buffered channel with capacity of 4
```

Concrete example of buffered channel to see what's going on.

```go
 package main

import (
	"bytes"
	"fmt"
	"os"
)

func main() {
	var stdoutBuff bytes.Buffer // create an in-memory buffer to help mitigate the nondeterministic nature of the output
	defer stdoutBuff.WriteTo(os.Stdout) // ensure that the buffer is written out to stdout before the process exits

	intStream := make(chan int, 4) // create a buffered channel with a capacity of four (4)
	go func() {
		defer close(intStream)
		defer fmt.Fprintln(&stdoutBuff, "Producer Done.")
		for i := 0; i < 5; i++ {
			fmt.Fprintf(&stdoutBuff, "Sending: %d\n", i)
			intStream <- i
		}
	}()

	for integer := range intStream {
		fmt.Fprintf(&stdoutBuff, "Received %v.\n", integer)
	}
}
// Output:
// Sending: 0
// Sending: 1
// Sending: 2
// Sending: 3
// Sending: 4
// Producer Done.
// Received 0.
// Received 1.
// Received 2.
// Received 3.
// Received 4.
```

Reading or writing to `nil` channels (the default value) will create a `deadlock`. Trying to close a `nil` channel will create a `panic` condition.

This is a reference for what the defined behaviour of working with channels is.

![Reference of Channels defined behaviour](/images/channel-behaviour-table.png)

### Unbuffered vs Buffered Channels

The only difference between `buffered` and `unbuffered channels` is an unbuffered channel is always declared with a capacity of zero (0). The buffered channel is declared with a known capacity, and useful in cases where the number of writes is known beforehand. One must be careful to always shoot for a buffered channel as it can be a premature optimization technicque that can lead to missed `datalocks`.

### The select Statement

The `select` statement is the glue to bind channels together; it's how we're able to compose channels together in a program to form larger abstractions. The `select` statement is one of the most crucial aspects of a `Go` program that uses concurrency.

How do `select` statements work, here's a quick simple example.

```go
package main

import (
	"fmt"
	"time"
)

func main() {
	start := time.Now()
	c := make(chan interface{})
	go func() {
		time.Sleep(5 * time.Second)
		close(c) // close the channel after waiting five (5) seconds
	}()

	fmt.Println("Blocking on read...")
	select {
	case <-c: // attempt to read on the channel
		fmt.Printf("Unblocked %v later.\n", time.Since(start))
	}
}
// Output:
// Blocking on read...
// Unblocked 5s later.
```

It looks like a `switch` block, but that's about it, the statements are not evaluated sequentially and there's no fallthrough if no match is found. All `channels` reads and writes are considered simultaneously to see if any of them are ready. The above example doesn't really require a `select` statement, it will be expanded on.