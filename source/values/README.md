The `values` package contains definitions of the `Value` type as used by the VM as the type of the elements of its `Mem` slice.

It is kept outside of the VM because it may be useful for optimization purposes to be able to import this as a `golang` import and if you put it in the VM that would involve importing the VM and also pretty much everything else (because of `exec` and `external`).

It contains all the non-native types that can serve as payloads for `Value`: `AbstractType`, `Map`, `Set`, `Snippet` and `Thunk`.

`Map` and `Set` were hacked together/cargo-culted from other people's code because I needed persistent data structures that would contain heterogeneous types and whereas I don't know how to hash all my types I do know how to compare any two values. They could presumably be improved by someone who knows what they're doing rewriting them from scratch on a totally different basis.

`AbstractType` is currently implemented as an ordered list of concrete types, for speed in iterating over it, but it will also eventually contain a slice of booleans for speed in checking membership.

## Files

* `abstract_type` contains the `AbstractType` type which serves as the payload for Pipefish's `TYPE` type.

* `map` contains the `Map` type which serves as the payload for Pipefish's `MAP` type.

* `set` contains the `Set` type which serves as the payload for Pipefish's `SET` type.

* `values_test` contains the tests for the package.

* `values` contains the definitions of the `Values` type and associated constants, and a comparison function used by the `Map` and `Set` types. It defines `Snippet` and `Thunk`, the payloads for the `SNIPPET` and the (subcucullar) `THUNK` type respectively.

