package engine

import "testing"

func TestSelectionString(t *testing.T) {
	sel := Selection{X1: 0, Y1: 64, Z1: 0, X2: 10, Y2: 70, Z2: 10}
	want := "(0, 64, 0) to (10, 70, 10)"
	if got := sel.String(); got != want {
		t.Errorf("Selection.String() = %q, want %q", got, want)
	}
}
