package dtypes_test

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/dtypes"
)

func TestStack(t *testing.T) {
	st := dtypes.NewStack[int]()
	_, ok := st.Pop()
	if ok {
		t.Fatal("Popped empty stack.")
	}
	_, ok = st.HeadValue()
	if ok {
		t.Fatal("Got head of empty stack.")
	}
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
	if st1.IsEmpty() {
		t.Fatal("Can't tell when set is empty.")
	}
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

func TestDigraph(t *testing.T) {
	g := dtypes.NewDigraph()
	dtypes.AddTransitiveArrow(g, "a", "b")
	dtypes.AddTransitiveArrow(g, "b", "c")
	dtypes.Add(g, "x")
	dtypes.AddTransitiveArrow(g, "x", "b")
	arrowsToC := dtypes.ArrowsTo(g, "c")
	if !arrowsToC.Contains("x") || !arrowsToC.Contains("a") || !arrowsToC.Contains("b") {
		t.Fatal("arrowsTo is broken.")
	}
	dtypes.Add(g, "q")
	dtypes.AddTransitiveArrow(g, "q", "z")
	dtypes.AddTransitiveArrow(g, "z", "q")
	result := dtypes.Tarjan(g)
	shouldBeC := result[0]
	if len(shouldBeC) != 1 {
		t.Fatal("Sort failed.")
	}
	if shouldBeC[0] != "c" {
		t.Fatal("Sort failed.")
	}
	shouldBeB := result[1]
	if len(shouldBeB) != 1 {
		t.Fatal("Sort failed.")
	}
	if shouldBeB[0] != "b" {
		t.Fatal("Sort failed.")
	}
	shouldBeA := result[2]
	if len(shouldBeA) != 1 {
		t.Fatal("Sort failed.")
	}
	if shouldBeA[0] != "a" {
		println(shouldBeA[0])
		t.Fatal("Sort failed.")
	}
	shouldBeX := result[3]
	if len(shouldBeX) != 1 {
		t.Fatal("Sort failed.")
	}
	if shouldBeX[0] != "x" {
		t.Fatal("Sort failed.")
	}
	shouldBeQAndZ := result[4]
	if len(shouldBeQAndZ) != 2 {
		t.Fatal("Sort failed.")
	}
	if shouldBeQAndZ[0] != "z" {
		t.Fatal("Sort failed.")
	}
	if shouldBeQAndZ[1] != "q" {
		t.Fatal("Sort failed.")
	}
}