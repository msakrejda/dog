package dogconf

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

/*

 [route all [get]]

 [route all [delete]]

get info on given route:

 [route 'route-id' [get]]

add a route:

 [route 'my-very-long-server-identifier-maybe-a-uuid'
   [create [addr='123.123.123.123:5432', user='foo', password='bar',dbname='x']]] 

patch a route:

 [route 'my-very-long-server-identifier-maybe-a-uuid' @ 5
   [patch [lock='t']]] 

delete a route:

 [route 'my-very-long-server-identifier-maybe-a-uuid' @ 5 [delete]]

*/

/*

grammar:

<request>    ::= "[" "route" <route-spec> "[" <command> "]" "]"
<route-spec> ::= "all" | <route-id>
<route-id>   ::= <identifier> "@" <ocn> | <identifier>
<command>    ::= <list-cmd> "[" <patch-list> "]" | <bare-cmd>
<bare-cmd>   ::= "get" | "delete"
<list-cmd>   ::= "patch" | "create"
<patch-list> ::= <patch> | <patch-list> "," <patch>
<patch>      ::= <identifier> "=" <value>
<value>      ::= <str-lit>
<ocn>        ::= <unsigned-integer>

*/

// Strip the quotes surrounding the string str and replace any escaped
// quotes by their actual values 
func stripStr(str string) string {
	// TODO: support quoted identifiers in addition to string literals
	l := len(str)
	if l < 2 || str[0] != '\'' || str[l-1] != '\'' {
		panic(fmt.Sprintf("Malformed string lexeme: %v", str))
	}
	stripped := strings.Replace(str[1:len(str)-1], "''", "'", -1)
	return stripped
}

func expect(s *Scanner, tokTyp TokenType) (*Token, error) {
	tok := s.Scan()
	if tok.Type != tokTyp {
		return nil, fmt.Errorf("Expected token '%v'; got '%v'",
			TokenTypeStr(tokTyp), tok)
	}
	return tok, nil
}

// Marker type as to be able to non-ambiguously recover a panic from a
// failed scan.  The scanner is included as to retain position
// information.
type ErrScanner struct {
	error
	BaseMessage string
	Scanner     *Scanner
}

func ParseRequest(r io.Reader) (rs *RequestSyntax, err error) {
	var s = new(Scanner)
	s.Init(r)

	// Convert only ErrScanner panics into regular return values.
	defer func() {
		if x := recover(); x != nil {
			if e, ok := x.(ErrScanner); ok {
				rs = nil
				err = e
			} else {
				panic(x)
			}
		}
	}()

	// Set up error handler for scanner.  This must be done
	// *after* Init() on the Scanner, or else it'll be overwritten
	// into oblivion.
	s.Error = func(s *Scanner, msg string) {
		// Blow up the entire scanning process if something
		// goes awry, at the very first incident.  It could be
		// useful to continue and accrue the errors rather
		// than blow up immediately, but future errors in the
		// lexer after a failed lex are questionable at best.
		pos := s.Position
		if !pos.IsValid() {
			pos = s.Pos()
		}

		embellished := fmt.Sprintf("%s: %s", pos, msg)
		panic(ErrScanner{
			error:       errors.New(embellished),
			BaseMessage: msg,
			Scanner:     s,
		})
	}

	return parseRequest(s)
}

func parseRequest(s *Scanner) (rs *RequestSyntax, err error) {
	_, err = expect(s, LBrace)
	if err != nil {
		return nil, err
	}

	tok, err := expect(s, Ident)
	if err != nil {
		return nil, err
	}
	if tok.Lexeme != "route" {
		return nil, fmt.Errorf("Expected 'route', got %v", tok)
	}

	spec, err := parseRouteSpec(s)
	if err != nil {
		return nil, err
	}

	// Only handle exactly one action per RequestSyntax for now
	action, err := parseAction(s)
	if err != nil {
		return nil, err
	}

	_, err = expect(s, RBrace)
	if err != nil {
		return nil, err
	}

	return &RequestSyntax{Spec: spec, Action: action}, nil
}

func parseRouteSpec(s *Scanner) (SpecSyntax, error) {
	// Here we either expect the keyword/identifier 'all' or a
	// quoted route identifier, optionally with an ocn.

	// If this is an 'all' specifier, short-circuit and return a
	// Syntax node for that.
	if tok := s.Peek(); tok.Type == Ident && tok.Lexeme == "all" {
		_, err := expect(s, Ident)
		if err != nil {
			return nil, err
		} else {
			return &TargetAllSpecSyntax{Target: tok}, nil
		}
	}

	// Both TargetOneSpecSyntax and TargetOcnSpecSyntax have a
	// targeted route.
	what, err := expect(s, String)
	if err != nil {
		return nil, err
	}

	// There may or may not be an OCN required for this command
	// and there may or may not be one present--we don't attempt
	// to resolve this at the grammar level.
	if tok := s.Peek(); tok.Type == At {
		_, err = expect(s, At)
		if err != nil {
			return nil, err
		}

		tok, err = expect(s, Int)
		if err != nil {
			return nil, err
		}

		var out TargetOcnSpecSyntax
		out.What = what
		out.Ocn = tok
		return &out, nil
	} else {
		// No OCN specified
		return &TargetOneSpecSyntax{What: what}, nil
	}

	panic("Uncovered conditions")
}

func parseAction(s *Scanner) (a ActionSyntax, err error) {
	_, err = expect(s, LBrace)
	if err != nil {
		return nil, err
	}

	tok, err := expect(s, Ident)
	if err != nil {
		return nil, err
	}

	switch tok.Lexeme {
	case "patch":
		props, err := parseProps(s)
		if err != nil {
			return nil, err
		}

		a = &PatchActionSyntax{Blamer: tok, PatchProps: props}
		goto out
	case "create":
		props, err := parseProps(s)
		if err != nil {
			return nil, err
		}

		a = &CreateActionSyntax{Blamer: tok, CreateProps: props}
		goto out
	case "get":
		a = &GetActionSyntax{Blamer: tok, GetToken: tok}
		goto out
	case "delete":
		a = &DeleteActionSyntax{Blamer: tok, DeleteToken: tok}
		goto out
	default:
		return nil, fmt.Errorf("Expected 'patch', 'create', "+
			"'get', or 'delete'; got %v", tok)
	}

	panic("Switch does not cover all cases when it should")

out:
	_, err = expect(s, RBrace)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Parses a series of tokens like:
//
//   [ ident = 'lit', ident2 = 'lit2' ]"
//
// Producing a token-to-token mapping as output.
func parseProps(s *Scanner) (map[*Token]*Token, error) {
	// Just advance over leading '['
	_, err := expect(s, LBrace)
	if err != nil {
		return nil, err
	}

	// The main routine: turning the token ident/literal mappings
	// into a more useful data structure.
	props := make(map[*Token]*Token)
	allowComma := false
	for tok := s.Peek(); tok.Type != RBrace; tok = s.Peek() {
		if allowComma && tok.Type == Comma {
			_, err := expect(s, Comma)
			if err != nil {
				return nil, err
			}
		}
		keyTok, err := expect(s, Ident)
		if err != nil {
			return nil, err
		}
		_, err = expect(s, Equals)
		if err != nil {
			return nil, err
		}
		valTok, err := expect(s, String)
		if err != nil {
			return nil, err
		}

		// Check validity of property keys being set.  This is
		// a convenience afforded by the fact that property
		// lists are only used by one construct in dogconf, so
		// it's possible to do some checking of the keys at
		// parse-time.  If this code needs be made
		// multi-purpose, it is best for validity-checking
		// code to move to the semantic analyzer.
		switch k := keyTok.Lexeme; k {
		case "addr", "lock", "dbnameIn", "dbnameRewritten":
			_, present := props[keyTok]
			if !present {
				props[keyTok] = valTok
			} else {
				return nil, fmt.Errorf("Duplicate key '%v' "+
					" in patch request", keyTok)
			}
		default:
			return nil, fmt.Errorf("Unknown key '%v': expected "+
				"'addr', 'lock', 'user', or 'password'", keyTok)
		}

		allowComma = true
	}

	// Just advance over trailing ']'
	_, err = expect(s, RBrace)
	if err != nil {
		return nil, err
	}

	return props, nil
}
