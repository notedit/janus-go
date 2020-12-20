# cameronelliott/janus-go

A Websocket transport Janus WebRTC package for Janus


## Goals

- First class stability and robustness
- Ability to easily reason about correctness of this library


## Intro

This is an experiment at creating an understandable, robust, Structured Concurrency based library for talking to Janus using Go
I am newer to structured concurrency, but I need a super super super robust and predictsble library for talking to Janus and this is my effort to have one
please join me, and provide any and all input and constructive criticism to help this become a more robust and trustworthy library


 
## Prinicpals guiding the effort to make this super trustworthy and stable


1. Be humble. I make mistakes, I invite and encourage your review and constructive critism on how to improve this lib.
1. Use a number of articles and blog posts as guides for the prinicpals behind how this library works.
1. Outline the techniques used to guide the code toward robustness and trustworthyness.
1. Provide a real world example of usage.


## Articles used to create the guidelines for stability and robustness

[Why You Should Be Using errgroup... Blum](https://bionic.fullstory.com/why-you-should-be-using-errgroup-withcontext-in-golang-server-handlers/)
    
[Managing Groups of Goroutines in Go. Block](https://medium.com/swlh/managing-groups-of-gorutines-in-go-ee7523e3eaca)

[Defer, Panic, and Recover/Gerrand](https://blog.golang.org/defer-panic-and-recover)


## Notable packages used 

[nhooyr/websocket](https://github.com/nhooyr/websocket) for websockets, the big win here IMHO is context.Context which supports cancellation.

## Programming Model
- One goroutine per websocket
- library users send pairs of (chan,msg) to that goroutine for commands to janus
- they then wait for a response on the channel they provided

## Websocket Setup Example

tx:= make(chan ToJanus)
createSession := make(chan chan Session)
done:= make(chan bool)
go JanusWSGoroutine(rx,tx, createSession)
wait done

## Session Creation Example

sessready := make(chan Session)
createSession <- sessready
if session,ok := <- sessready; !ok { 

  return
}



## Handle Example

hanready := make(chan Handle)
session.makeHandle <- hanready
handle := <- hanready



## Plugin Command Example

## Techniques used for the stability and robustness of this project

1. Goroutine lifetimes should be nested. Structured Concurrency
1. [Share by communicating](https://golang.org/doc/effective_go.html#sharing)
1. Use Panic and Recover for unexpected handable errors
1. Writers to channels should defer to close channels
1. Use context.Context on call paths from Http requests (see google blog post below)
1. Use a lint or static analysis tool to make sure there are no missed errors
1. Use best practices for error handling and panicing. Dave Cheyney has good srticles on this.




## Google blog post about usage of context.context inside request handlers

An old Google blog post said this about their policy about context.Context:
> At Google, we require that Go programmers pass a Context parameter as the first argument to every function on the call path between incoming and outgoing requests. This allows Go code developed by many different teams to interoperate well. It provides simple control over timeouts and cancelation and ensures that critical values like security credentials transit Go programs properly.
[Go Google Blog](https://blog.golang.org/context)


### Credits 
Original work by [https://github.com/notedit](Notedit)
In addition to this popular library, he has a lot
of cool WebRTC related software, and stuff for the Medooze media server, another awesome
SFU like Janus. Check his stuff out! :smile:








