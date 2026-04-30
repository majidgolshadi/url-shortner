package domain

// Range is a pre-allocated block of integer IDs assigned exclusively to one node,
// avoiding per-request DB coordination for ID generation.
type Range struct {
	Start uint
	End   uint
}
