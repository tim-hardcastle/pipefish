Besides the test files, the important stuff here is:

* `hub.go`
* `hub.pf`
* `hab snap`

`hub.pf` is the Pipefish service that the user talks to when they talk to the hub. `hub.go` actually inmplements it.

Much of the logic has to be in `hub.go`, with `hub.pf` a thin skin over the top, because there are lots of things that can only be done from Go and not from Pipefish, e.g. compiling and running Pipefish services. A few things would be better off in `hub.pf` but are in `hub.go` for historical reasons.

The way it works is that the `hub.pf` file has a type `Hub` wrapping `io.Writer`, and a variable initialized as `HUB Hub? = NULL`. On creation of the `hub` service, we inject an `io.Writer` into the variable, where the `Write` method tells the `hub.pf` to do the thing in question.

(The reason we do it this way rather than injecting the Go `*hub.Hub` object itself is that this requires the production of an `.so` file which takes over a minute to compile and is larger than the main executable.)

`hub snap` contains logic for writing tests in the REPL.