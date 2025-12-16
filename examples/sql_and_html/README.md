This demo comes in three parts.

* `crud.pf` is the main program. It sets up a very simple SQL daatabase and supplies commands to interact with it. It is purely imperative, because all it's doing besides defining types is IO.

* By contrast, `htmlFormat` is purely functional, a little library for rendering Pipefish values into HTML, knocked up for the occasion to demonstrate how we can use "snippets" to embed a new or existing DSL in Pipefish.

* `client` uses `crud` as an external service, communicating with it (in this case) over http, so it can be run on localhost.