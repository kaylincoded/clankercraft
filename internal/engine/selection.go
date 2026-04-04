package engine

import "fmt"

// Selection represents a WorldEdit-style two-corner region selection.
type Selection struct {
	X1, Y1, Z1 int
	X2, Y2, Z2 int
}

// String returns a human-readable representation of the selection.
func (s Selection) String() string {
	return fmt.Sprintf("(%d, %d, %d) to (%d, %d, %d)", s.X1, s.Y1, s.Z1, s.X2, s.Y2, s.Z2)
}
