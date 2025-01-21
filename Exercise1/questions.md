Exercise 1 - Theory questions
-----------------------------

### Concepts

What is the difference between *concurrency* and *parallelism*?
> Concurrency is using one resource to juggle multiple tasks / threads, switching between them efficiently. Parallelism on the other hand, is using multiple resources to execute tasks at (literally) the same time.

What is the difference between a *race condition* and a *data race*? 
> A race condition occurs when the of the outcome of something is being determined by the order of execution, when the order is non-deterministic. A data race occurs when instructions from different threads access the same place in memory (with at least one of them changing the value located there), without anything deterministically choosing which instruction / which thread should be executed first.
 
*Very* roughly - what does a *scheduler* do, and how does it do it?
> A scheduler restricts the non-determinism in a real-time system, by ordering the use of the system resources and assigning them to the appropriate threads.


### Engineering

Why would we use multiple threads? What kinds of problems do threads solve?
> Threads help us perform multiple tasks at once (or at least concurrently), enabling multitasking and preventing individual processes / tasks from blocking the execution of other tasks.

Some languages support "fibers" (sometimes called "green threads") or "coroutines"? What are they, and why would we rather use them over threads?
> Fibers use cooperative multitasking in contrast to the preemptive multitasking threads use. Since fibers are implicitly synchronised, we avoid many of the issues of thread safety.

Does creating concurrent programs make the programmer's life easier? Harder? Maybe both?
> *Your answer here*

What do you think is best - *shared variables* or *message passing*?
> *Your answer here*


