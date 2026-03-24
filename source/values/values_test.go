package values_test

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/values"
)

func TestSet(t *testing.T) {
	s1 := values.Set{}
	s2 := s1.Add(values.FALSE)
	if s1.Len() != 0 {
		t.Fatal("So much for PDS.")
	}
	if s2.Len() != 1 {
		t.Fatal("Add failed.")
	}
	s3 := s2.Add(values.FALSE)
	if s3.Len() != 1 {
		t.Fatal("Sets aren't meant to do that.")
	}
	s4 := s3.Add(values.TRUE)
	if s4.Len() != 2 {
		t.Fatal("Your sets aren't settin' properly.")
	}
	s5 := s1.Add(values.ONE).Add(values.FALSE).Add(values.OK)
	if s5.Len() != 3 {
		t.Fatal("Your sets aren't settin' properly.")
	}
	s6 := s4.Union(s5)
	if s6.Len() != 4 {
		t.Fatal("Add failed.")
	}
	sl := s6.AsSlice()
	if len(sl) != 4 {
		t.Fatal("AsSlice failed.")
	}
	s7 := s4.Subtract(s5)
	if s7.Len() != 1 {
		t.Fatal("Subtract failed.")
	}
	s8 := s5.Subtract(s4)
	if s8.Len() != 2 {
		t.Fatal("Subtract failed.")
	}
	if !s8.Contains(values.ONE) || !s8.Contains(values.OK) {
		t.Fatal("Subtract failed.")
	}
	s9 := s5.Intersect(s4)
	if s9.Len() != 1 {
		t.Fatal("Intersect failed.")
	}
	if !s9.Contains(values.FALSE) {
		t.Fatal("Intersect failed.")
	}
	numbers := values.Set{}.Add(values.Value{values.INT, 1}).
	Add(values.Value{values.INT, 2}).Add(values.Value{values.INT, 3})
	sum := 0 
	numbers.Range(func(element values.Value){sum = sum + element.V.(int)})
	if sum != 6 {
		t.Fatal("Range failed.")
	}
}