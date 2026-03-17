This module declares a `Service` struct which wraps around the compiler/vm, hiding their internals.

This allows someone who imports this module into their own program to do anything to/with a service that the user can via the hub. This is enforced by the fact that the hub itself only sees the `Service` type and is ignorant of the compiler, vm, parser, etc.

This does mean that the API is rather large and somewhat cumbersome, because bits have been added on ad hoc to meet the specific needs of whatever it is I wanted the hub to do that week. It is also more unstable than most of the project.