This implements a few useful datatypes: `Set` to wrap around the native `map`; `Stack` to wra around lists, and `Digraph`, for doing Tarjan sorts, which is supported by an imported `OrderedMap` type and an `OrderedSet` to go with it.

The digraph is deterministic because it's useful for both me and the user if the initialization of a script is in a fixed order.