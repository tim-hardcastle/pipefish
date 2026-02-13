The `values` package contains definitions of the `Value` type as used by the VM as the type of the elements of its `Mem` slice.

It is kept outside of the VM because it may be useful for optimization purposes to be able to import this as a `golang` import and if you put it in the VM that would involve importing the VM and also pretty much everything else (because of `exec` and `external`).

This package also contains some structures in the `map`, `set`, and `iterator` files that serve as the payload for some `Value`s.

`iterator` supplies the payload for the (user-invisible) `ITERATOR` type which we use under the hood to power `for` loops.

`map` and `set` are the payloads for the `MAP` and `SET` types. They were hacked together/cargo-culted from other people's code because I needed persistent data structures that would contain heterogeneous types and whereas I don't know how to hash all my types I do know how to compare any two values (modulo the usual corner cases such as functions). They could presumably be improved by someone who knows what they're doing rewriting them from scratch on a totally different basis.
