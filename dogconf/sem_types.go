// Semantic Analyzer Types
//
// For mechanics, see sem.go
package dogconf

// Union of types that describe a kind of target for an action
type Target interface {
	Blamer
}

// Targets everything.  Useful with delete and get.
type TargetAll TargetAllSpecSyntax

// Targets a specific record, regardless of OCN -- hence, subject to
// race conditions.
type TargetOne struct {
	Blamer
	What string
}

// The most specific target: targets a specific thing at a specific
// version, as to be able to raise optimistic concurrency violations
// when there is a version/ocn mismatch.
type TargetOcn struct {
	Blamer
	TargetOne
	Ocn uint64
}

// Toplevel emission from semantic analysis: a single semantically
// analyzed action to be interpreted by the executor.
type Directive interface {
	Blamer
}

type PatchDirective struct {
	Blamer
	TargetOcn
	Attrs map[*Token]Token
}

type CreateDirective struct {
	TargetOne
	Attrs map[*Token]Token
}

type DeleteDirective struct {
	// Only valid targets for delete: 'all' and targets with ocn.
	Target
}

type GetDirective struct {
	// Only valid targets for get: 'all' and targets without ocn
	Target
}

type AttrChange struct {
}
