// Semantic Analyzer for dogconf
//
// Converts AST into structures that might be able to be applied onto
// a run-time state.  They are still subject to error, such as an OCN
// clash, and that can only be resolved at execution time.
package dogconf

import (
	"fmt"
)

// Returned when a wrong-in-all-situations type of target is included
// with an action.  For example: a 'patch' request without an
// OCN-augmented target, or a 'create' given the 'all' target.
type ErrBadTarget struct {
	error
}

// Return error values decorated with token positioning information
func semErrf(blam Blamer, format string, args ...interface{}) error {
	return fmt.Errorf("%s: %s",
		blam.Blame().Pos,
		fmt.Sprintf(format, args))
}

func Analyze(req *RequestSyntax) (Directive, error) {
	switch req.Action.(type) {
	case *PatchActionSyntax:
		return nil, nil
	case *CreateActionSyntax:
		//		return analyzeCreate(req, a)
		return nil, nil
	case *GetActionSyntax:
		//		return analyzeGet(req, a)
		return nil, nil
	case *DeleteActionSyntax:
		//		return analyzeDelete(req, a)
		return nil, nil
	}

	panic(fmt.Errorf("Attempting to semantically analyze "+
		"un-enumerated action type %T", req.Action))
}
