package dtypes_test

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/dtypes"
)

func TestStack(t *testing.T) {
	st := dtypes.NewStack[int]()
	st.Push(6)
	st.Push(7)
	st.Push(8)
	st.Push(9)
	if st.Len() != 4 {
		t.Fatal("Stack has wrong length.")
	}
	st2 := st.Copy()
	i := st2.Find(8)
	if i != 1 {
		t.Fatal("Find failed.")
	}
	i = st2.Find(86)
	if i != -1 {
		t.Fatal("Find succeeded.")
	}
	h, ok := st2.HeadValue()
	if !ok {
		t.Fatal("Couldn't find head.")
	}
	if h != 9 {
		t.Fatal("Wrong head value.")
	}
	h, ok = st2.Pop()
	if !ok {
		t.Fatal("Couldn't find head.")
	}
	if h != 9 {
		t.Fatal("Wrong head value.")
	} 
	h, ok = st2.HeadValue()
	if !ok {
		t.Fatal("Couldn't find head.")
	}
	if h != 8 {
		t.Fatal("Wrong head value.")
	}
	L, ok := st2.Take(2)
	if !ok {
		t.Fatal("Couldn't take two elements.")
	}
	if len(L) != 2 {
		t.Fatal("Took wrong number of elements.")
	}
	if L[0] != 7 || L[1] != 8 {
		t.Fatal("Took wrong elements.")
	}
	h, ok = st2.HeadValue()
	if !ok {
		t.Fatal("Couldn't find head.")
	}
	if h != 6 {
		t.Fatal("Wrong head value.")
	}
	_, ok = st2.Take(2)
	if ok {
		t.Fatal("Took too many elements.")
	}
}

func TestSet(t *testing.T) {
	st1 := dtypes.From(6, 7, 8, 9)
	st2 := dtypes.MakeFromSlice([]int{6, 7, 8}).Add(9)
	st3 := dtypes.MakeFromSlice([]int{3, 4, 5, 6, 7})
	if len(st1) != 4 {
		t.Fatal("Can't construct sets.")
	}
	if len(st3) != 5 {
		t.Fatal("Can't construct sets.")
	}
	if !st1.Contains(8) {
		t.Fatal("Can't construct sets.")
	}
	if !st2.Contains(8) {
		t.Fatal("Can't construct sets.")
	}
	if st3.Contains(8) {
		t.Fatal("Can't construct sets.")
	}
	if len(st1) != 4 {
		t.Fatal("Can't construct sets.")
	}
	if !st2.OverlapsWith(st3) {
		t.Fatal("Can't overlap sets.")
	}
	if len(st2.SubtractSet(st3)) != 2 {
		t.Fatal("Can't suubtract sets.")
	}
	st2.AddSet(st3)
	if len(st2) != 7 {
		t.Fatal("Can't add sets.")
	}
	sl := st2.ToSlice()
	if len(sl) != 7 {
		t.Fatal("Can't convert set to slice.")
	}
}