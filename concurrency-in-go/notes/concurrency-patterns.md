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

## Confinement

Idea of ensuring information is only ever available from _one_ concurrent process. When this is achieved, a program is considered concurrency safe and no _synchronization_ is needed.

### Ad-hoc

**Ad-hoc** confinement is when you achieve confinement through a convention, whether it be set by the languages community, the group you work with or the codebase. Keeping convention can be difficult without static analysis tools of code before commit.

```go
package main

import (
	"fmt"
)

func main() {
	data := make([]int, 4)

	loopData := func(handleData chan<- int) {
		defer close(handleData)
		for i := range data {
			handleData <- data[i]
		}
	}

	handleData := make(chan int)
	go loopData(handleData)

	for num := range handleData {
		fmt.Println(num)
	}
}
// Output:
// 0
// 0
// 0
// 0
```

In the above example, `data` slice of integers is available from both the `loopData` function and the loop over the `handleData` channel; however, by convention we're only accessing it from the `loopData` function.

### Lexical

**Lexical** confinement involves using lexical scope to expose only the correct data and concurrency primitives for multiple concurrent processes to use. Exposing `read` or `write` only channels is a way of lexical confinement, it's impossible to do the wrong thing on the code.

example revisit

```go
package main

import (
	"fmt"
)

func main() {
	chanOwner := func() <-chan int {
		results := make(chan int, 5) // 1: instantiate the channel within the lexical scope of the chanOwner function.
		go func() {
			defer close(results)
			for i := 0; i <= 5; i++ {
				results <- i
			}
		}()
		return results
	}

	consumer := func(results <-chan int) { // 3: receive a read-only copy of an int channel
		for result := range results {
			fmt.Printf("Received: %d\n", result)
		}
		fmt.Println("Done receiving!")
	}

	results := chanOwner() // 2: receive the read aspect of the channel and we're able to pass it to the consumer
	consumer(results)
}
// Output:
// Received: 0
// Received: 1
// Received: 2
// Received: 3
// Received: 4
// Received: 5
// Done receiving!
```

Confinement can improve performance and reduce cognitive load on developers.

### The for-select Loop

The `for-select` loop is extremely popular in `Go` programs, at it's simplest it's something like this...

```go
for { // either loop infinitely or range over something
    select {
        case [match]:
            // do things
    }
}
```

This pattern may show up in a couple different scenarios..

- Sending iteration variables out on a channel.
  - Oftentimes you'll want ot convert something that can be iterated over into values on a channel.

```go
for _, s := range []string{"a", "b", "c"} {
    select {
        case <- done:
          return
        case stringStream <- s:
    }
}
```

- Looping infinitely waiting to be stopped
  - It's very common to create `goroutines` that loop infinitely until they're stopped.

First:

```go
for {
    select {
        case <- done:
            return
        default:
    }

    // Do non-preemptable work
}
```

If the `done` channel isn't closed, we'll exit the select statement and continue on to the rest of the `for` loop.

Second:

```go
for {
    select {
        case <- done:
            return
        default:
            // Do non-preemptable work
    }
}
```

### Preventing Goroutine Leaks

`Goroutines` are not garbage collected and no matter how small a footprint, it's a bad idea to let them run rampant. There are a few paths to `goroutine` termination.

- When it has completed it's work
- When it cannot continue it's work due to an unrecoverable error
- When it's told to stop working (cancellation)

The first two (2) are free as part of our algorithm. What about `cancellation`? Cancellation can be accomplished by establishing a signal between the parent `goroutine` and the child `goroutines`. This by convention is a `read-only` channel named `done`.

```go
package main

import (
	"fmt"
	"time"
)

func main() {
    // 1: pass the done channel to doWork
	doWork := func(done <-chan interface{}, strings <-chan string) <-chan interface{} {
		terminated := make(chan interface{})
		go func() {
			defer fmt.Println("doWork exited.")
			defer close(terminated)
			for {
				select {
				case s := <-strings:
					// Do something interesting
					fmt.Println(s)
				case <-done: // 2: for-select pattern, one case checks to see if done is closed
					return
				}
			}
		}()
		return terminated
	}

	done := make(chan interface{})
	terminated := doWork(done, nil)

	go func() { // 3: create another goroutine that will cancel the goroutine spawned in doWork
		// Cancel the operation after 1 second.
		time.Sleep(1 * time.Second)
		fmt.Println("Canceling doWork goroutine...")
		close(done)
	}()

	<-terminated // 4: join the goroutine spawned from doWork with the main goroutine
	fmt.Println("Done.")
}
// Output:
// Canceling doWork goroutine...
// doWork exited.
// Done.
```

### The or-channel

Combining one or more `done` channels into a single `done` channel. It's acceptable to do this with `select` statement; however, sometimes you can't know the number of `done` channels you're working with. This pattern creates a composite `done` channel through recursion and goroutines.

```go
package main

func main() {
	var or func(channels ...<-chan interface{}) <-chan interface{}
	or = func(channels ...<-chan interface{}) <-chan interface{} {

		switch len(channels) {
		case 0:

			return nil
		case 1:

			return channels[0]
		}

		orDone := make(chan interface{})
		go func() {

			defer close(orDone)

			switch len(channels) {
			case 2:

				select {
				case <-channels[0]:
				case <-channels[1]:
				}
			default:

				select {
				case <-channels[0]:
				case <-channels[1]:
				case <-channels[2]:
				case <-or(append(channels[3:], orDone)...):

				}
			}
		}()
		return orDone
	}
}
```

## Error Handling

Who should be responsible for handling an error? When does the error stop ferrying up the stack? Who/What is responsible for this? As with everything else in _concurrent_ programming, this becomes a little more complex. Because concurrent functions are running independent of their parent or siblings, it can be difficult to reason about.

This is an example.

```go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	checkStatus := func(
		done <-chan interface{},
		urls ...string,
	) <-chan *http.Response {
		responses := make(chan *http.Response)
		go func() {
			defer close(responses)
			for _, url := range urls {
				resp, err := http.Get(url)
				if err != nil {
					fmt.Println(err) // #1
					continue
				}
				select {
				case <-done:
					return
				case responses <- resp:
				}
			}
		}()
		return responses
	}

	done := make(chan interface{})
	defer close(done)

	urls := []string{"https://www.google.com", "https://badhost"}
	for response := range checkStatus(done, urls...) {
		fmt.Printf("Response: %v\n", response.Status)
	}
}
// Output:
// Get https://www.google.com: dial tcp: Protocol not available
// Get https://badhost: dial tcp: Protocol not available
//
// 1: Here we see the goroutine doing its best to signal that there's an error.
// What else can it do? It cannot pass it back! How many errors is too many?
// Does it continue making requests?
```

The above example puts the `goroutine` in an awkward position by printing the error and hoping someone is listening. The concerns should be separated and the error passed to another part of the program that has more complete information.

Correct:

```go
package main

import (
	"fmt"
	"net/http"
)

func main() {
	type Result struct { // 1: create a type that encompasses both the *http.Response and the error possible
		Error    error
		Response *http.Response
    }
    // 2: returns a channel that can be read from to retrieve results of an iteration of the loop
	checkStatus := func(done <-chan interface{}, urls ...string) <-chan Result {
		results := make(chan Result)
		go func() {
			defer close(results)

			for _, url := range urls {
				var result Result
                resp, err := http.Get(url)
                // 3: create a Result instance with the Error and Response fields set
				result = Result{Error: err, Response: resp}
				select {
				case <-done:
					return
				case results <- result: // 4: write the Result to our channel
				}
			}
		}()
		return results
	}
	done := make(chan interface{})
	defer close(done)

	urls := []string{"https://www.google.com", "https://badhost"}
	for result := range checkStatus(done, urls...) {
        // 5: deal with errors coming out of the goroutine started by checkStatus intelligently,
        // and with the full context of the larger program
		if result.Error != nil {
			fmt.Printf("error: %v\n", result.Error)
			continue
		}
		fmt.Printf("Response: %v\n", result.Response.Status)
	}
}
// Output:
// Response: 200 OK
// error: Get https://badhost: dial tcp: lookup badhost: Temporary failure in name resolution
```

## Pipelines

A `pipeline` is just another tool to form abstractions in the systems|program. It is specially useful when a program needs to process streams or batches of data. A series of things that take data in, perform an operations on it, and pass that data back out. Like _functions_ or _struct_, pipelines separate areas of concern.

For functions to be considered part of a `pipeline` stage, that stage consumes and returns the same type. A stage must be reified by the language so that it may be passed around.

Example pipeline:

```go
package main

import (
	"fmt"
)

func main() {

    // multiply consumes and returns an int slice
	multiply := func(values []int, multiplier int) []int {
		multipliedValues := make([]int, len(values))
		for i, v := range values {
			multipliedValues[i] = v * multiplier
		}
		return multipliedValues
	}

    // add consumes and returns an int slice
	add := func(values []int, additive int) []int {
		addedValues := make([]int, len(values))
		for i, v := range values {
			addedValues[i] = v + additive
		}
		return addedValues
	}

    // consume both multiply and add without modifying the code.
    // calls from in-to-out as in multiply -> add -> multiply
	ints := []int{1, 2, 3, 4}
	for _, v := range multiply(add(multiply(ints, 2), 1), 2) {
		fmt.Println(v)
	}
}
```

### Best Practices for Constructing Pipelines

`Channels` are uniquely qualified to constructing piplines in `Go` because they fulfill all the basic requirements. They can receive and emit values, they can safely be used concurrently, they can be ranged over, and they are reified by the language.

Same example using `channels`:

```go
package main

import (
	"fmt"
)

func main() {

    // generator takes in a variadic slice of integers, constructs a buffered channel of integers
    // with a length equal the incoming integer slice, starts a goroutine, and returns the constructed channel.
    // On the goroutine that was created, generator ranges over the variadic slice that was passed in
    // and sends the slices' values on the channel it created.
	generator := func(done <-chan interface{}, integers ...int) <-chan int {
		intStream := make(chan int)
		go func() {
			defer close(intStream)
			for _, i := range integers {
				select {
				case <-done:
					return
				case intStream <- i:
				}
			}
		}()
		return intStream
	}

	multiply := func(done <-chan interface{}, intStream <-chan int, multiplier int) <-chan int {
		multipliedStream := make(chan int)
		go func() {
			defer close(multipliedStream)
			for i := range intStream {
				select {
				case <-done:
					return
				case multipliedStream <- i * multiplier:
				}
			}
		}()
		return multipliedStream
	}

	add := func(done <-chan interface{}, intStream <-chan int, additive int) <-chan int {
		addedStream := make(chan int)
		go func() {
			defer close(addedStream)
			for i := range intStream {
				select {
				case <-done:
					return
				case addedStream <- i + additive:
				}
			}
		}()
		return addedStream
	}

    // define a done channel to signal when a goroutine should terminate
	done := make(chan interface{})
	defer close(done)

	// run generator and convert concrete values into stream of data
	intStream := generator(done, 1, 2, 3, 4)
	pipeline := multiply(done, add(done, multiply(done, intStream, 2), 1), 2)

	for v := range pipeline {
		fmt.Println(v)
	}
}
// Output:
// 6
// 10
// 14
// 18
```

This looks like a lotmore code for replicating the same example. What exactly is there to gain? First, we’re using channels. This is obvious but significant because it allows two things: at the end of our pipeline, we can use a range statement to extract the values, and at each stage we can safely execute concurrently because our inputs and outputs are safe in concurrent contexts. **Second** each stage of the pipeline is executing concurrently. This means that any stage only need wait for its inputs, and to be able to send its outputs.

### Handy Generators

#### Repeat

```go
// repeat repeats the values passed into it infinitely until told to stop
func repeat(done <-chan interface{}, values ...interface{}) <-chan interface{} {
	valueStream := make(chan interface{})
	go func() {
		defer close(valueStream)

		for {
			for _, v := range values {
				select {
					case <-done:
					return
					case valueStream <- v:
				}
			}
		}
	}()

	return valueStream
}
```

```go
// take the first num items off of its incoming valueStream and then exit
func take(done <-chan interface{}, valueStream <-chan interface{}, num int) <-chan interface{} {
	takeStream := make(chan interface{})

	go func() {
		defer close(takeStream)

		for i := 0; i < num; i++ {
			select {
				case <- done:
				return
				case takeStream <- <- valueStream:
			}
		}
	}()

	return takeStream
}

// repeatFn is a repeating generator, that repeatedly calls a function fn
func repeatFn(done <-chan interface{}, fn func() interface{}) <-chan interface{} {
    valueStream := make(chan interface{})
    go func() {
        defer close(valueStream)
        for {
            select {
            case <-done:
            	return
            case valueStream <- fn():
            }
        }
    }()
    return valueStream
}


func toString(
	done <-chan interface{},
	valueStream <-chan interface{},
) <-chan string {
	stringStream := make(chan string)
	go func() {
		defer close(stringStream)
		for v := range valueStream {
			select {
			case <-done:
				return
			case stringStream <- v.(string):
			}
		}
	}()
	return stringStream
}


func toInt( done <-chan interface{}, valueStream <-chan interface{}) <-chan int {
	intStream := make(chan int)
	go func() {
		defer close(intStream)
		for v := range valueStream {
			select {
			case <-done:
				return
			case intStream <- v.(int):
			}
		}
	}()
	return intStream
}


func primeFinder(done <-chan interface{}, intStream <-chan interface{}) <-chan interface{} {
	primeStream := make(chan interface{})
	go func() {
		defer close(primeStream)
		for integer := range intStream {
			integer -= 1
			prime := true
			for divisor := integer - 1; divisor > 1; divisor-- {
				if integer%divisor == 0 {
					prime = false
					break
				}
			}

			if prime {
				select {
				case <-done:
					return
				case primeStream <- integer:
				}
			}
		}
	}()
	return primeStream
}

func main() {
	done := make(chan interface{})
    defer close(done)

    rand := func() interface{} { return rand.Int()}

    for num := range take(done, repeatFn(done, rand), 10) {
        fmt.Println(num)
    }
}
```

### Benchmark Tests

The below tests are dependent on the above pipelines.

```go
package main

import "testing"

func BenchmarkGeneric(b *testing.B) {
	done := make(chan interface{})
	defer close(done)

	b.ResetTimer()
	for range toString(done, take(done, repeat(done, "a"), b.N)) {
	}
}

func BenchmarkTyped(b *testing.B) {
	repeat := func(done <-chan interface{}, values ...string) <-chan string {
		valueStream := make(chan string)
		go func() {
			defer close(valueStream)
			for {
				for _, v := range values {
					select {
					case <-done:
						return
					case valueStream <- v:
					}
				}
			}
		}()
		return valueStream
	}

	take := func(
		done <-chan interface{},
		valueStream <-chan string,
		num int,
	) <-chan string {
		takeStream := make(chan string)
		go func() {
			defer close(takeStream)
			for i := num; i > 0 || i == -1; {
				if i != -1 {
					i--
				}
				select {
				case <-done:
					return
				case takeStream <- <-valueStream:
				}
			}
		}()
		return takeStream
	}

	done := make(chan interface{})
	defer close(done)

	b.ResetTimer()
	for range take(done, repeat(done, "a"), b.N) {
	}
}
```

## Fan-Out, Fan-In

**Fan-out, Fan-in** is the ability to reuse a single stage of our pipeline on multiple goroutines in an attempt to parallelize pulls from an upstream stage. _Fan-out_ is a term to describe the process of starting multiple goroutines to handle input from the pipeline, and _fan-in_ is a term to describe the process of combining multiple results into one channel. For a stage to take advantage of these stages, the following needs to appply...

- It doesn't rely on values that the stage had calculated before
- It takes a long time to run

Here's a naive implementation of a prime finder to later showcase the power of `fan-out, fan-in`.

```go
package main

import (
	"fmt"
	"math/rand"
	"time"
)

func take(done <-chan interface{}, valueStream <-chan interface{}, num int) <-chan interface{} {
	takeStream := make(chan interface{})

	go func() {
		defer close(takeStream)

		for i := 0; i < num; i++ {
			select {
			case <-done:
				return
			case takeStream <- <-valueStream:
			}
		}
	}()

	return takeStream
}

func repeatFn(done <-chan interface{}, fn func() interface{}) <-chan interface{} {
	valueStream := make(chan interface{})
	go func() {
		defer close(valueStream)
		for {
			select {
			case <-done:
				return
			case valueStream <- fn():
			}
		}
	}()
	return valueStream
}

func toString(
	done <-chan interface{},
	valueStream <-chan interface{},
) <-chan string {
	stringStream := make(chan string)
	go func() {
		defer close(stringStream)
		for v := range valueStream {
			select {
			case <-done:
				return
			case stringStream <- v.(string):
			}
		}
	}()
	return stringStream
}

func toInt(done <-chan interface{}, valueStream <-chan interface{}) <-chan int {
	intStream := make(chan int)
	go func() {
		defer close(intStream)
		for v := range valueStream {
			select {
			case <-done:
				return
			case intStream <- v.(int):
			}
		}
	}()
	return intStream
}

func primeFinder(done <-chan interface{}, intStream <-chan int) <-chan interface{} {
	primeStream := make(chan interface{})
	go func() {
		defer close(primeStream)
		for integer := range intStream {
			integer -= 1
			prime := true
			for divisor := integer - 1; divisor > 1; divisor-- {
				if integer%divisor == 0 {
					prime = false
					break
				}
			}

			if prime {
				select {
				case <-done:
					return
				case primeStream <- integer:
				}
			}
		}
	}()
	return primeStream
}

func main() {
	rand := func() interface{} { return rand.Intn(50000000) }

	done := make(chan interface{})
	defer close(done)

	start := time.Now()

	randIntStream := toInt(done, repeatFn(done, rand))
	fmt.Println("Primes:")
	for prime := range take(done, primeFinder(done, randIntStream), 10) {
		fmt.Printf("\t%d\n", prime)
	}

	fmt.Printf("Search took: %v", time.Since(start))
}
// Output:
// Primes:
//         24941317
//         36122539
//         6410693
//         10128161
//         25511527
//         2107939
//         14004383
//         7190363
//         45931967
//         2393161
// Search took: 28.825832209
```

As you can see, it took almost **29 seconds** to find 10 prime numbers. Here's the same example using the `fan-out, fan-in` process.

```go
package main

import (
	"fmt"
	"math/rand"
	"time"
)

func take(done <-chan interface{}, valueStream <-chan interface{}, num int) <-chan interface{} {
	takeStream := make(chan interface{})

	go func() {
		defer close(takeStream)

		for i := 0; i < num; i++ {
			select {
			case <-done:
				return
			case takeStream <- <-valueStream:
			}
		}
	}()

	return takeStream
}

func repeatFn(done <-chan interface{}, fn func() interface{}) <-chan interface{} {
	valueStream := make(chan interface{})
	go func() {
		defer close(valueStream)
		for {
			select {
			case <-done:
				return
			case valueStream <- fn():
			}
		}
	}()
	return valueStream
}

func toString(
	done <-chan interface{},
	valueStream <-chan interface{},
) <-chan string {
	stringStream := make(chan string)
	go func() {
		defer close(stringStream)
		for v := range valueStream {
			select {
			case <-done:
				return
			case stringStream <- v.(string):
			}
		}
	}()
	return stringStream
}

func toInt(done <-chan interface{}, valueStream <-chan interface{}) <-chan int {
	intStream := make(chan int)
	go func() {
		defer close(intStream)
		for v := range valueStream {
			select {
			case <-done:
				return
			case intStream <- v.(int):
			}
		}
	}()
	return intStream
}

func primeFinder(done <-chan interface{}, intStream <-chan int) <-chan interface{} {
	primeStream := make(chan interface{})
	go func() {
		defer close(primeStream)
		for integer := range intStream {
			integer -= 1
			prime := true
			for divisor := integer - 1; divisor > 1; divisor-- {
				if integer%divisor == 0 {
					prime = false
					break
				}
			}

			if prime {
				select {
				case <-done:
					return
				case primeStream <- integer:
				}
			}
		}
	}()
	return primeStream
}


func fanIn(done <-chan interface{}, channels ...<-chan interface{}) <-chan interface{} {
	var wg sync.WaitGroup
	multiplexedStream := make(chan interface{})

	multiplex := func(c <-chan interface{}) {
		defer wg.Done()
		for i := range c {
			select {
			case <-done:
				return
			case multiplexedStream <- i:
			}
		}
	}

	// Select from all the channels
	wg.Add(len(channels))
	for _, c := range channels {
		go multiplex(c)
	}

	// Wait for all the reads to complete
	go func() {
		wg.Wait()
		close(multiplexedStream)
	}()

	return multiplexedStream
}

func main() {
	rand := func() interface{} { return rand.Intn(50000000) }

	done := make(chan interface{})
	defer close(done)

	start := time.Now()

	randIntStream := toInt(done, repeatFn(done, rand))
	numFinders := runtime.NumCPU()
	fmt.Printf("Spinning up %d prime finders.\n", numFinders)
	finders := make([]<-chan interface{}, numFinders)
	fmt.Println("Primes:")
	for i := 0; i < numFinders; i++ {
		finders[i] = primeFinder(done, randIntStream)
	}

	for prime := range take(done, fanIn(done, finders...), 10) {
		fmt.Printf("\t%d\n", prime)
	}

	fmt.Printf("Search took: %v", time.Since(start))
}
// Output:
// Spinning up 12 prime finders.
// Primes:
//         6410693
//         10128161
//         24941317
//         36122539
//         25511527
//         2107939
//         14004383
//         7190363
//         2393161
//         45931967
// Search took: 2.999153121
```

This search took orders of magnitude less time, up to 14 times faster. If we concentrate on the algorithm to find the primes it can be even faster. The point of this though was to optimize a slow stage by using `fan-out, fan-in`.

## The or-done-channel
