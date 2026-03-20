The initializer, compiler, and VM are mainly tested by getting them to initialized a Pipefish script, and then evaluate an expression as though it had been input into the REPL.

These are the scripts they use to initialize from. In many cases their use for testing the thing they say they're testing will be far from evident unless you also look at the line of code to be evaluated.