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


My plan is to implement a (small portion of) some classic text-based adventure game, Zork or ADVENT or whatever, in (a cut-down version of) Cognate, which I will write in Pipefish.

Pipefish is my own language, an attempt, so far successful, to write a functional language that you can really hack stuff out in, with a special orientation to CRUD apps, middleware, microservices and DSLs. To see what it looks like:


* The lexer for this jam.
* A little text-based adventure game in Pipefish. 
* A little demo of a CRUD app to demonstrate how we do DSLs (in this case HTML and SQL).
* A little text-munging program used as part of my tooling for developing Pipefish.
* And the wiki. https://github.com/tim-hardcastle/Pipefish/wiki

It is pretty much feature-complete, but rquires some thorough testing, a fuzzer, optimization, more tooling, a few more standard libraries, etc.

Cognate is the brainchild of a guy who goes by Stavromula on the internet. Like a million other people he thought "What if Forth was also Lisp?" but he made it work. Here is some Cognate:

```
Def Factor (Zero? Modulo Swap);

Def Primes (
	Fold (
		Let I be our potential prime;
		Let Primes be the found primes;
		Let To-check be Take-while (<= Sqrt I) Primes;
		When None (Factor of I) To-check
			(Append List (I)) to Primes;
	) from List () over Range from 2
);

Print Primes up to 1000;
```
I have three purposes in this besides having fun.

* Pipefish is long overdue for some heavy dogfooding. It still has a tendency to crash when trying to compile/exeecute *malformed* code, when the user strays off the happy path. Ergonomic improvements will occur to me. I'll see the typos in the error messages.

The second is to show off both Pipefish and Cognate. Each is in its own way a charming and original language.

* The third is that I'd like to do a full implementation of Cognate eventually and this will be a good start.

For this last reason, although I'm not going to do all the tricky parts of the syntax and semantics, I *am* going to be doing the very hardest part (essentially, closures) although I won't need them to write an adventure game. Because retrofitting them would be such a PITA.









In the first version the actual underlying syntax and semantics are being disguised by writing the `~` and the `R` identifier without whitespace between them, but with or without whitespace a token boundary is there, and they are two different identifiers.


Variables and functions exist in the scope of the block in which they are defined, and may be declared in any order. This allows, for example, for binary recursion.



// We won't usually want to see all the fields, so we overload the string function.
string(t Token) -> string : 
    t[val] in null :
        string t[tokenType]
    else :
        (string t[tokenType]) + "(" + (string t[val]) + ")"




Doing this sort of thing is not encouraged.



The gods Šamaš (and) Adad] placed at my disposal the lore of the diviner, a craft that cannot be changed; the god Marduk, the sage of the gods, granted me a broad mind (and) extensive knowledge as a gift; the god Nabû, the scribe of everything, bestowed on me the precepts of his craft as a present; the gods Ninurta and Nergal endowed my body with power, virility, and unrivalled strength. I learned the craft of the sage Adapa, the secret and hidden lore of all of the scribal arts. I am able to recognize celestial and terrestrial omens and can discuss them in an assembly of scholars. I am capable of arguing with expert diviners about the series “If the liver is a mirror image of the heavens.” I can resolve complex mathematical divisions and multiplications that do not have an easy solution. I have read cunningly written texts in obscure Sumerian and Akkadian that are difficult to interpret. I have carefully examined inscriptions on stone from before the Deluge that are sealed, stopped up, and confused.


, no, you were wrong about everything. It started off with "funny how you don't know anyone who's had COVID", until we all did know someone who'd had COVID. Then it was "funny how you don't know anyone who's died of COVID", until we all did. Then it was "it's killed less people than the flu", until it killed more. And "fearmonger Fauci" because he said as many as 200,000 would die, which turned out to be an underestmate, he wasn't mongering enough fear. Then it was: "The lockdowns will never end, bro, it's all about control, bro" and "the mask mandates will never end, it's all about control, bro, trust me, bro". Then it was "miCrOcHiPs iN tHe vAcCiNes", and guess what, there were no microchips in the vaccines. And then it was "tHe vAcCiNeS wiLL kiLL eVeryOne ... annnnny day now", and the vaccinated continue not only to be hundreds of times less likely to die of COVID, but are healthier across the board.



 but you know perfectly well that you haven't done the math, and that you're just reciting the party line. (While complaining about "useful idiots".
If you look at all the other first world countries, not only is universal healthcare cheaper per capita than our dumpster fire, it also costs less per capita of taxpayers' money. We're actually paying more in taxes to prop up our system of Potemkin capitalism and pretend we have a free market system, then they are to cover everyone and have no premiums or copays.
Can you think of any particular thing that would make America incapable of doing what everyone else can do? (I mean, apart from the Republican Party?)




, I also think that stupid superstitions were enforced by men. But no-one (I've ever heard) claims that there's a special paternal instinct that allows fathers to diagnose and cure there children's diseases, so I didn't mention that.
The history of dentistry tells us that the nonexistence of toothworms and that sugar causes caries was discovered by a guy with a microscope. His name was Pierre Fauchard.
There's a lot of bad things done by Big Pharma. The book "Bad Science: Quacks, Hacks, and Big Pharma Flacks" by Ben Goldacre does a good all-round job of showing up Bad Medicine, I recommend it.
Individual doctors and nurses, not so much. Remember they're people like you and me who just took a different major in college. Most of them did so with the idea that healing people is a noble profession. Same with medical researchers. They *want* to cure cancer and heart disease, 'cos they all know people who've died of those things, and also there's kudos in winning Nobel Prizes and stuff. There's no point in their education where the professor takes them aside and says "I know you went into scientific research to discover the truth and help people, but actually what we do is conceal the truth and make people ill, for money. Promise not to tell?"
In the UK, a general practitioner (primary physician if you're American) is paid for how many patients they have registered with them, not per visit. A doctor who keeps their patients healthier can therefore either take on more patients, thereby getting richer, or they can spend more time on the golf course or whatever, so their incentives align with their mission.


It's 



If you want a meme about how the second law of thermodynamics proves that the Earth is flat, you go to a flat-Earther. But if you want someonewho can use the laws of thermodynamics to design a fidge or an engine, you go to a globetard.

If you want a meme saying you should ignore the evidence of your own eyes if it contradicts the flat arth, because of perspe ctive, you go to a flat-Earther. If you want someone to use the laws of perspective to make a VR headset or a flight simulator or a computer game, you go to a globetard.

If you want a meme saying that gravity doesn't exist, you go to a flat-Earther. If you want someone who'll use the law of gravity to design a bridge that will stay up or a plane that will fly, you go to a globetard.

If you want a meme about "dEnSiTy aNd bUoYanCy", you go to a flat-Earther. If you want someone to design a ship that won't capsize, you go to a globetard.

If you want a meme about how seeing the moon in daytime proves the Earth is flat, you go to a flat-Earther. If you want to predict a solar or lunar eclipse to the minute, you go to a globetard.

Etc, etc. If you're the ones who understand how things work, we is it that the globetards are the people who actually make things that work, and all you make is memes?






You may remember 


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


