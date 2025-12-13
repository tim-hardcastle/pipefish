# Tokenization rules

Here we give the exact rules for how to form a Pipefish identifier — the sequence of characters which names a variable, a function, a type, a struct's field, etc.

We can divide Unicode characters into the following groups:

* Whitespace. You can't use this in an identifier.

* The "protected punctuation", the characters `(`, `)`, `[`, `]`, `{`, `}`, `,`, `;`, `:`, `.`, `"`, `` ` ``, `'` and `|`. You can't use any of these in an identifier.

* Alphabetic characters. You can use any of these in an identifier.

* The numerals `0` ... `9`. You can use these in an identifier, but you can't use them to start an identifier.

* Symbols, consisting of everything else except the underscore character _. You can use these in an identifier.

* The underscore, `_`. You can use this in an identifier, but you can't use it at the start or the end of an identifier (except in the special case of `_` being the whole of the identifier).

The rule governing their arrangement is that alphabetic characters can only go next to other alphabetic characters or the underscore; numerals can only go next to other numerals or the underscore; and symbols can only go next to other symbols or the underscore.

So the underscore acts as a bridge between the three other kinds of characters you can put in an identifier: foo_+ is a legal identifier; foo+ is not; &_3 is a legal identifier but &3 is not.

The upshot of this is that if people want to write a/b or x+1 they can do so and this is unambiguous, since the letters and numbers must belong to different identifiers than the symbols. *However*, it may sometimes be useful to qualify a symbol by a letter or word or vice-versa. At this point one can write for example `~_R`, using the `_` as a neutral bridge between symbol and letter.

## Note

A little experimentation may make you think that you can break these rules. For example if you define :

```
def

(x int) ~R (y int) :
    x == -y 
```

then this will compile and work *almost* exactly how you think it will. However, you have *not* in this case defined an infix operator `~R`: rather, the definition above is equivalent to:

```
def

(x int) ~ R (y int) :
    x == -y 
```

In the first version the actual underlying syntax and semantics are being disguised by writing the `~` and the `R` identifier without whitespace between them, but with or without whitespace a token boundary is there, and they are two different identifiers.





Doing this sort of thing is not encouraged.



The gods Šamaš (and) Adad] placed at my disposal the lore of the diviner, a craft that cannot be changed; the god Marduk, the sage of the gods, granted me a broad mind (and) extensive knowledge as a gift; the god Nabû, the scribe of everything, bestowed on me the precepts of his craft as a present; the gods Ninurta and Nergal endowed my body with power, virility, and unrivalled strength. I learned the craft of the sage Adapa, the secret and hidden lore of all of the scribal arts. I am able to recognize celestial and terrestrial omens and can discuss them in an assembly of scholars. I am capable of arguing with expert diviners about the series “If the liver is a mirror image of the heavens.” I can resolve complex mathematical divisions and multiplications that do not have an easy solution. I have read cunningly written texts in obscure Sumerian and Akkadian that are difficult to interpret. I have carefully examined inscriptions on stone from before the Deluge that are sealed, stopped up, and confused.


, no, you were wrong about everything. It started off with "funny how you don't know anyone who's had COVID", until we all did know someone who'd had COVID. Then it was "funny how you don't know anyone who's died of COVID", until we all did. Then it was "it's killed less people than the flu", until it killed more. And "fearmonger Fauci" because he said as many as 200,000 would die, which turned out to be an underestmate, he wasn't mongering enough fear. Then it was: "The lockdowns will never end, bro, it's all about control, bro" and "the mask mandates will never end, it's all about control, bro, trust me, bro". Then it was "miCrOcHiPs iN tHe vAcCiNes", and guess what, there were no microchips in the vaccines. And then it was "tHe vAcCiNeS wiLL kiLL eVeryOne ... annnnny day now", and the vaccinated continue not only to be hundreds of times less likely to die of COVID, but are healthier across the board.







import

"strings"

cmd

solve :
    get input from File("/Users/tobe/code/aoc2025/input.txt")
    instructions = parseInput input
    post partOne(instructions)

def

parseInput(i) :
    from pairs = [] for _::l = range strings.split(strings.trim(i, " \n"), "\n") :
        pairs + [l[0]::l[1::len(l)]]

partOne(moves) :
    second from pos, zeroes = 50, 0 for _::move = range moves :
        newPos, zeroes + (newPos == 0 : 1; else : 0)
    given:
        newPos = move[0] == "R" :
            (pos + move[1]) mod 100
        else :
            (pos - move[1]) mod 100


