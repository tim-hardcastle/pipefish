The `compiler`, `parser`, `initializer`, and `vm` tests have some functions and data structures in common; as do the tests for `hub` and `pf`.

The `test_helper` package collects this shared logic, and a few things which aren't shared but belong here thematically.