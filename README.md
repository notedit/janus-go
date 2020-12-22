# cameronelliott/janus-go

A Go language Websocket based package for controlling the Janus WebRTC gateway

## TL;DR

The main thrust of this package is to improve on reliability, stability, and trustworthiness of the original repository.
  
Please join me by providing feedback and Pull Requests



## Prinicpals guiding the effort to make this super trustworthy and stable


1. Be humble. I make mistakes, I invite and encourage your review and constructive critism on how to improve this lib.
1. Use a number of articles and blog posts as guides for the prinicpals behind how this library works.
1. Outline the techniques used to guide the code toward robustness and trustworthyness.
1. Provide a real world example of usage.


## Articles used to create the guidelines for stability and robustness

[Why You Should Be Using errgroup... Blum](https://bionic.fullstory.com/why-you-should-be-using-errgroup-withcontext-in-golang-server-handlers/)
    
[Managing Groups of Goroutines in Go. Block](https://medium.com/swlh/managing-groups-of-gorutines-in-go-ee7523e3eaca)

[Defer, Panic, and Recover/Gerrand](https://blog.golang.org/defer-panic-and-recover)


## Notable packages 

[nhooyr/websocket](https://github.com/nhooyr/websocket): used for websockets, the big win here IMHO is context.Context which supports cancellation.

[golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup): may or may not be included on this package in the future, but can provide a very useful tool for propagating error values between goroutines and their creators.


## Techniques used for the stability and robustness of this project

1. Switch from github.com/gorilla/websocket to github.com/nhooyr.io/websocket, the reason being is that nhooyr.io supports context.Context for many websocket operations. This is a big win for reliable concurrency in many situations. See comment/link below about Google policy requiring use of context.Context
1. Propagate all non-nil `error`s up the call stack, *AND* up the goroutine chain. This should mean no ignored, lost, swallowed errors. 
1. Support the use of this package [golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup). This is how error values can be returned from goroutines to their parents. Links to articles about Structured Concurrency below.
1. Writers to channels should use `defer close(channelx)` to close channels, especially where nested goroutine lifetimes occur. (Structured Conncurrency)
1. Use context.Context on call paths from Http requests (see google blog post below)
1. Use a lint or static analysis tool to make sure there are no missed errors
1. Use best practices for logging levels. Dave Cheyney has good articles on this.


## Techniques not used
1. [Share by communicating](https://golang.org/doc/effective_go.html#sharing),  
this is a great approach for robust concurrency, 
but it requires more changes to the original than I am comfortable making.
It would also break the API for existing code.
So we are going to live with the mutexs and locking in the code.


## Google blog post about usage of context.context inside request handlers

An old Google blog post said this about their policy about context.Context:
> At Google, we require that Go programmers pass a Context parameter as the first argument to every function on the call path between incoming and outgoing requests. This allows Go code developed by many different teams to interoperate well. It provides simple control over timeouts and cancelation and ensures that critical values like security credentials transit Go programs properly.
[Go Google Blog](https://blog.golang.org/context)


### Credits 
Original work by [https://github.com/notedit](Notedit)
In addition to this popular library, he has a lot
of cool WebRTC related software, and stuff for the Medooze media server, another awesome
SFU like Janus. Check his stuff out! :smile:








