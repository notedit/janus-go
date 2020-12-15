# janus-go

A Websocket transport Janus WebRTC package for Go
Created by [https://github.com/notedit](Notedit),
in addition to this popular library, he has a lot
of cool programs for the Medooze media server, another awesome
SFU like Janus. Check his stuff out! :smile:

I really appreciate this fanastic package from Notedit.
In working with it, I decided to expiriment with some changes
to make it work better with the style of programming I am exploring.

Basically, I am 


Here are the changes I am working on:
1. Switch to [nhooyr/websocket](https://github.com/nhooyr/websocket) for websockets, the big win here IMHO is context.Context support which better supports [structured concurrency](https://medium.com/swlh/managing-groups-of-gorutines-in-go-ee7523e3eaca) which in theory can help concurrency robustness when done well.
- Hope to remove goroutine creation in the library, I usually prefer to do this as a library's caller, as it makes it easier for me to think/reason about thread/goroutine issues
- Hope to remove chan creation in the library, again I like to create these externally to libraries I use if possible, it can make thinking/reasoning about robustness easier for me.
- Plan to use a lint or static analysis tool to make sure there are no missed errors
- While already mentioned, I am removing the errors channels, which have been previously used to signal errors, and switching to the old-fashioned method of returning from functions on unhandlable errors.


I'm just getting started, let's see what happens.







