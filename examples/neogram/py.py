# A tiny functional Pratt parser.
#
# Input is a list of token strings, e.g.
#
#     ["abs", "-", "3", "+", "2", "*", "4", "!"]
#
# Each parser returns
#
#     (tree, remaining_tokens)

INFIX_BP = {
    "+": 10,
    "-": 10,
    "*": 20,
}

PREFIX_BP = 30
POSTFIX_BP = 40


def parseAll(tokens):
    tree, tokens = parseExpression(tokens, 0)

    if tokens:
        raise SyntaxError(f"Unexpected token {tokens[0]}")

    return tree


def parseExpression(tokens, minimum_bp):
    # Parse something that can begin an expression,
    # then extend it with postfix and infix operators.
    return parseInfix(*parsePrefix(tokens), minimum_bp)


def parsePrefix(tokens):
    token = tokens[0]
    tokens = tokens[1:]

    # Parenthesised expression.
    if token == "(":
        tree, tokens = parseExpression(tokens, 0)

        if not tokens or tokens[0] != ")":
            raise SyntaxError("Expected ')'")

        return tree, tokens[1:]

    # Unary minus.
    if token == "-":
        tree, tokens = parseExpression(tokens, PREFIX_BP)
        return ("neg", tree), tokens

    # abs operator.
    if token == "abs":
        tree, tokens = parseExpression(tokens, PREFIX_BP)
        return ("abs", tree), tokens

    # Otherwise it must be a number.
    return ("num", int(token)), tokens


def parseInfix(left, tokens, minimum_bp):
    if not tokens:
        return left, tokens

    token = tokens[0]

    # Postfix factorial.
    if token == "!":
        if POSTFIX_BP < minimum_bp:
            return left, tokens

        return parseInfix(
            ("fact", left),
            tokens[1:],
            minimum_bp,
        )

    # Stop if this isn't an infix operator.
    if token not in INFIX_BP:
        return left, tokens

    bp = INFIX_BP[token]

    if bp < minimum_bp:
        return left, tokens

    right, tokens = parseExpression(tokens[1:], bp)

    return parseInfix(
        (token, left, right),
        tokens,
        minimum_bp,
    )


# ------------------------------------------------------------
# Test

tokens = ["abs", "-", "3", "+", "2", "*", "4", "!"]

tree = parseAll(tokens)

print(tree)