package set

import (
	"testing"
)

func TestSet(t *testing.T) {
	set1 := New[int](1, 2, 3, 4, 5, 6, 7)
	set2 := New[int](3, 4, 5, 6, 7, 8, 9)

	if !set1.Difference(set2).Equal(New[int](1, 2, 8, 9)) {
		t.Errorf("difference should be {1, 2, 8, 9}")
	}

	if !set1.Union(set2).Equal(New[int](1, 2, 3, 4, 5, 6, 7, 8, 9)) {
		t.Errorf("union should be {1, 2, 3, 4, 5, 6, 7, 8, 9}")
	}

	if !set1.Intersection(set2).Equal(New[int](3, 4, 5, 6, 7)) {
		t.Errorf("intersection should be {3, 4, 5, 6, 7}")
	}

	set1.Remove(3, 4, 5)
	if !set1.Equal(New[int](1, 2, 6, 7)) {
		t.Errorf("remove should leave set as {1, 2, 6, 7}")
	}

	set1.Remove(3, 4, 5)
	if !set1.Equal(New[int](1, 2, 6, 7)) {
		t.Errorf("set should be {1, 2, 6, 7} after remove")
	}

	if !set1.Clear().Empty() {
		t.Errorf("clear should leave set empty}")
	}

	if !set2.Contains(9) {
		t.Errorf("set should contain 9")
	}

	if set2.Contains(1) {
		t.Errorf("set should not contain 1")
	}
}
