// Contains operators used for blaming syntax nodes for problems.
package dogconf

// Interface for types that can identify one token to place error
// position at.
type Blamer interface {
	Blame() *Token
}

func (t *Token) Blame() *Token {
	return t
}

func (t *TargetAllSpecSyntax) Blame() *Token {
	return t.Target
}

func (t *TargetOneSpecSyntax) Blame() *Token {
	return t.What
}

func (t *TargetOcnSpecSyntax) Blame() *Token {
	return t.What
}
