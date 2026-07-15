This document explains for people already familiar with Go what relationship Pipefish has to it.

## Aims of Pipefish

First of all, it is not intended as a replacement language, like Kotlin is for Java; nor on the other hand as a merely auxiliary language to be used when Go needs a scripting language --- though it *can* be used for that purpose. Rather, it bears the same sort of relationship to Go as Python does to C: a separate stand-alone language which benefits from the existence of a compiled language and its standard and third-party libraries to supply it with speed and an ecosystem.

Pipefish is more dynamic than Go, more declarative, more high-level, and will inherently always be a little slower. You wouldn't want to use it where Go really shines, at big infrastructure projects: rather, it allows rapid development of CRUD apps, middleware, distributed services, and DSLs to wrap them in, in ways that compete with PHP for smaller projects and Java for larger ones.

## Go interop

Pipefish has been designed from the beginning to be highly compatible with Go: the reference implementation runs on a Go VM. It has lexical Go interop: you can write a Pipefish function with a Go body.

Because I don't own Google, you can't *lexically* embed Pipefish in Go in the same way, but you can import the `pf` library into your Go project and use it like you would a library for embedded Python or whatever.

Since it's so easy to embed Go into Pipefish, it's usually trivial and sometimes automatable to wrap a Go library in a Pipefish library. Any library in the Go ecosystem is at most a few hours away from being part of the Pipefish ecosystem. 

## Syntax and semantics

The design of Pipefish has followed Go where this was sensible, Python as a fallback option, and actual originality only when necessary. (It has often turned out to be necessary.) Some examples:

* `foo(x int, y, z bool)` is a good function signature in Pipefish like it is in Go.

* The literals are what you're used to, e.g. booleans are `true` and `false`, the int type is called `int`, the string type is called `string`, `0xFF` is a hexadecimal integer, etc.

* The standard libraries almost all have the same names as the equivalent Go libraries, and contain functions with the same names where appropriate.

* The useful `main`/`init` distinction is preserved: `init` is executed as soon as the module it's in is compiled.

* Like Go, Pipefish has a nominal type system with no inheritance.

* Clone types are like Go's assigned types, but better.

* Pipefish has ad-hoc interfaces that are even more ad-hoc than Go's interfaces.

* And also generics right now rather than in ten years. (Anyone who would like Pipefish to be even more like Go can wait ten years before using them.)

* The `for` loops are very similar to Go despite technically being expressions.

* Everything is 0-indexed; all ranges are from-including-to-excluding.

## Design philosophy

Pipefish shares Go's philosophy of being a small boring language learnable in a weekend where everyone can read everyone else's code because everyone's code is just using `for` loops over and over on a small set of built-in types.

Like Go, the goal is a language that is permanently in version 1.x, and if anything more suspicious of additions. We will soon be able to say: "That's enough core language" and freeze development.

Even more than Go, Pipefish is about developer experience and rapid development. Go has fast compile times for rapid iteration; Pipefish has livecoding and a REPL. Go made tests into a first-class feature of the language; Pipefish allows you to intersperse them among the code you're writing. Logging statements take everything you like about just sticking `println` in your code and make it as powerful as a debugger. Etc, etc.

## Fish

Pipefish is named after a fish and so is Rob Pike.
