package values_test

import (
	"testing"

	"github.com/tim-hardcastle/pipefish/source/values"
)

func TestAbstractType(t *testing.T) {
	numbers := values.AbT(values.FLOAT, values.INT)
	if numbers.Len() != 2 {
		t.Fatal("Can't do type creation.")
	}
	if ! numbers.Contains(values.INT) {
		t.Fatal("Can't do type creation.")
	}
	containers := values.AbT(values.LIST, values.PAIR, values.MAP)
	if numbers.Equals(containers) {
		t.Fatal("Can't check equality.")
	}
	both := numbers.Union(containers)
	if both.Len() != 5 {
		t.Fatal("Can't do union.")
	}
	bothAgain := containers.Union(numbers)
	if ! bothAgain.Equals(both) {
		t.Fatal("Union isn't commutative.")
	}
	if ! numbers.IsSubtypeOf(both) {
		t.Fatal("Can't do union.")
	}
	if ! containers.IsProperSubtypeOf(both) {
		t.Fatal("Can't do union.")
	}
	if ! both.IsSubtypeOf(both) {
		t.Fatal("Can't do subtypes.")
	}
	if both.IsProperSubtypeOf(both) {
		t.Fatal("Can't do proper subtypes.")
	}
	if both.IsSubtypeOf(numbers) {
		t.Fatal("Can't do subtypes.")
	}
	if both.IsProperSubtypeOf(numbers) {
		t.Fatal("Can't do proper subtypes.")
	}
	bothAndString := both.Insert(values.STRING)
	if !bothAndString.Contains(values.STRING) {
		t.Fatal("Can't insert.")
	}
	bothAndStringAndString := bothAndString.Insert(values.STRING)
	if !bothAndStringAndString.Equals(bothAndString) {
		t.Fatal("Insert is broken.")
	}
	withoutStringAgain := bothAndString.Without(values.AbT(values.STRING))
	if withoutStringAgain.Contains(values.STRING) {
		t.Fatal("Can't do without.")
	}
	if ! withoutStringAgain.Equals(both) {
		t.Fatal("Can't do equals.")
	}
	numbersAndNull := numbers.Insert(values.NULL)
	if ! (numbersAndNull.Intersect(bothAndString)).Equals(numbers) {
		t.Fatal("Can't do intersection.")
	}
	if ! (numbersAndNull.PartlyIntersects((bothAndString))) {
		t.Fatal("Can't check partial intersection.")
	}
	if  (numbersAndNull.PartlyIntersects((containers))) {
		t.Fatal("Can't check partial intersection.")
	}
	if numbersAndNull.Equals(containers) {
		t.Fatal("Can't check equality.")
	}
	if ! numbers.Without(values.AbT(values.INT)).Is(values.FLOAT) {
		t.Fatal("Can't check Is.")
	}
	if numbers.Is(values.FLOAT) {
		t.Fatal("Can't check Is.")
	}
	if ! (values.AbT(values.ValueType(7), values.ValueType(6)).String() == "[6, 7]") {
		println(values.AbT(values.ValueType(7), values.ValueType(6)).String())
		t.Fatal("Can't make string.")
	}
}

func TestCompare(t *testing.T) {
	if ! (values.Value{values.FLOAT, 0.2}).Compare(values.Value{values.FLOAT, 0.3}) {
		t.Fatal("Float comparison failed.")
	}
	if (values.Value{values.FLOAT, 0.2}).Compare(values.Value{values.FLOAT, 0.2}) {
		t.Fatal("Float comparison failed.")
	}
	if ! (values.Value{values.ValueType(99), []values.Value{{values.FLOAT, 0.3}}}). 
			Compare(values.Value{values.ValueType(99), []values.Value{{values.FLOAT, 99.9}}}) {
		t.Fatal("Struct comparison failed.")
	}
	if ! (values.Value{values.ValueType(99), []values.Value{{values.FLOAT, 0.3}}}). 
			Compare(values.Value{values.ValueType(99), []values.Value{{values.FLOAT, 99.9}, {values.STRING, "aardvark"}}}) {
		t.Fatal("Struct comparison failed.")
	}
	if (values.Value{values.ValueType(99), []values.Value{{values.FLOAT, 0.3}}}). 
			Compare(values.Value{values.ValueType(99), []values.Value{{values.FLOAT, 0.3}}}) {
		t.Fatal("Struct comparison failed.")
	}
	if ! (values.Value{values.STRING, "aardvark"}).Compare(values.Value{values.STRING, "zebra"}) {
		t.Fatal("String comparison failed.")
	}
	if (values.Value{values.STRING, "aardvark"}).Compare(values.Value{values.STRING, "aardvark"}) {
		t.Fatal("String comparison failed.")
	}
	if ! (values.Value{values.TYPE, values.AbT(values.ValueType(7))}).Compare(values.Value{values.TYPE, values.AbT(values.ValueType(8))}) {
		t.Fatal("Type comparison failed.")
	}
	if ! (values.Value{values.TYPE, values.AbT(values.ValueType(7))}).Compare(values.Value{values.TYPE, values.AbT(values.ValueType(7), values.ValueType(8))}) {
		t.Fatal("Type comparison failed.")
	}
	if ! (values.Value{values.TYPE, values.AbT(values.ValueType(7), values.ValueType(6))}).Compare(values.Value{values.TYPE, values.AbT(values.ValueType(7), values.ValueType(8))}) {
		t.Fatal("Type comparison failed.")
	}
	if ! (values.Value{values.TYPE, values.AbT()}).Compare(values.Value{values.TYPE, values.AbT(values.ValueType(7), values.ValueType(8))}) {
		t.Fatal("Type comparison failed.")
	}
}

func TestMap(t *testing.T) {
	m1 := values.Map{}
	m2 := m1.Set(values.FALSE, values.OK)
	if m1.Len() != 0 {
		t.Fatal("So much for PDS.")
	}
	if m2.Len() != 1 {
		t.Fatal("Set failed.")
	}
	el, ok := m2.Get(values.FALSE)
	if !(ok && el.T == values.SUCCESSFUL_VALUE && el.V == nil) {
		t.Fatal("Get failed.")
	}
	m3 := m2.Set(values.TRUE, values.ONE)
	if m2.Len() != 1 {
		t.Fatal("So much for PDS.")
	}
	if m3.Len() != 2 {
		t.Fatal("Set failed.")
	}
	el, ok = m3.Get(values.TRUE)
	if !(ok && el.T == values.INT && el.V.(int) == 1) {
		t.Fatal("Get failed.")
	}
	m4 := m3.Set(values.TRUE, values.ONE)
	if m4.Len() != 2 {
		t.Fatal("Maps shouldn't work like that.")
	}
	m5 := m3.Delete(values.FALSE)
	if m5.Len() != 1 {
		t.Fatal("Delete failed.")
	}
	_, ok = m5.Get(values.FALSE)
	if ok {
		t.Fatal("Delete failed.")
	}
	numbers := values.Map{}.Set(values.Value{values.RUNE, 'a'}, values.Value{values.INT, 1}).
		Set(values.Value{values.RUNE, 'b'}, values.Value{values.INT, 2}).
		Set(values.Value{values.RUNE, 'c'}, values.Value{values.INT, 3}).
		Set(values.Value{values.RUNE, 'd'}, values.Value{values.INT, 4})
	sum := 0
	numbers.Range(func(key, value values.Value) { sum = sum + value.V.(int) })
	if sum != 10 {
		t.Fatal("Range failed.")
	}
	empty := numbers.Delete(values.Value{values.RUNE, 'c'}).
		Delete(values.Value{values.RUNE, 'b'}).
		Delete(values.Value{values.RUNE, 'a'}).
		Delete(values.Value{values.RUNE, 'd'})
	if empty.Len() != 0 {
		t.Fatal("Delete failed")
	}
	slice := numbers.AsSlice()
	sum = 0
	for _, v := range slice {
		sum = sum + v.Val.V.(int)
	}
	if sum != 10 {
		t.Fatal("AsSlice failed.")
	}
	numbers2 := values.Map{}.Set(values.Value{values.INT, 1}, values.Value{values.RUNE, 'a'}).
		Set(values.Value{values.INT, 2}, values.Value{values.RUNE, 'a'}).
		Set(values.Value{values.INT, 3}, values.Value{values.RUNE, 'a'})	
	vec := numbers2.KeysAsVector()
	sum = 0
	for i := range vec.Len() {
		val, _ := vec.Index(i)
		j := val.(values.Value).V.(int)
		sum = sum + j
	}
	if sum != 6 {
		t.Fatal("AsSlice failed.")
	}
}

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
	numbers.Range(func(element values.Value) { sum = sum + element.V.(int) })
	if sum != 6 {
		t.Fatal("Range failed.")
	}
}
