package vm

import (
	"unicode/utf8"

	"github.com/tim-hardcastle/pipefish/source/values"
	"src.elv.sh/pkg/persistent/vector"
)

type Iterator interface {
	Unfinished() bool
	GetKey() values.Value
	GetValue() values.Value
	GetKeyValuePair() (values.Value, values.Value)
}

type DecIterator struct { // For an 'x::y' range, going down.
	StartVal int
	Val      int
	MinVal   int
	pos      int
}

func (it *DecIterator) Unfinished() bool {
	return it.Val >= it.MinVal
}

func (it *DecIterator) GetKey() values.Value {
	keyResult := values.Value{values.INT, it.pos}
	it.pos++
	it.Val--
	return keyResult
}

func (it *DecIterator) GetValue() values.Value {
	valResult := values.Value{values.INT, it.Val}
	it.pos++
	it.Val--
	return valResult
}

func (it *DecIterator) GetKeyValuePair() (values.Value, values.Value) {
	keyResult := values.Value{values.INT, it.pos}
	valResult := values.Value{values.INT, it.Val}
	it.pos++
	it.Val--
	return keyResult, valResult
}

type EnumIterator struct { // For an 'x::y' range, going down.
	Type values.ValueType
	Max  int
	pos  int
}

func (it *EnumIterator) Unfinished() bool {
	return it.pos < it.Max
}

func (it *EnumIterator) GetKey() values.Value {
	panic("This doesn't happen.") // If we just want to iterate over the keys of a list, we use an KeyIncIterator.

}

func (it *EnumIterator) GetValue() values.Value {
	valResult := values.Value{it.Type, it.pos}
	it.pos++
	return valResult
}

func (it *EnumIterator) GetKeyValuePair() (values.Value, values.Value) {
	keyResult := values.Value{values.INT, it.pos}
	valResult := values.Value{it.Type, it.pos}
	it.pos++
	return keyResult, valResult
}

type IncIterator struct { // For an 'x::y' range, going up.
	StartVal int
	Val      int
	MaxVal   int
	pos      int
}

func (it *IncIterator) Unfinished() bool {
	return it.Val < it.MaxVal
}

func (it *IncIterator) GetKey() values.Value {
	keyResult := values.Value{values.INT, it.pos}
	it.pos++
	it.Val++
	return keyResult
}

func (it *IncIterator) GetValue() values.Value {
	valResult := values.Value{values.INT, it.Val}
	it.pos++
	it.Val++
	return valResult
}

func (it *IncIterator) GetKeyValuePair() (values.Value, values.Value) {
	keyResult := values.Value{values.INT, it.pos}
	valResult := values.Value{values.INT, it.Val}
	it.pos++
	it.Val++
	return keyResult, valResult
}

// This is for the case when we ask to range over the key only of something which has an integer key.
type KeyIncIterator struct {
	Max int
	pos int
}

func (it *KeyIncIterator) Unfinished() bool {
	return it.pos < it.Max
}

func (it *KeyIncIterator) GetKey() values.Value {
	keyResult := values.Value{values.INT, it.pos}
	it.pos++
	return keyResult
}

func (it *KeyIncIterator) GetValue() values.Value {
	panic("KeyIncIterator returns only keys.")
}

func (it *KeyIncIterator) GetKeyValuePair() (values.Value, values.Value) {
	panic("KeyIncIterator returns only keys.")
}

type ListIterator struct {
	VecIt vector.Iterator
	pos   int
}

func (it *ListIterator) Unfinished() bool {
	return it.VecIt.HasElem()
}

func (it *ListIterator) GetKey() values.Value {
	panic("This doesn't happen.") // If we just want to iterate over the keys of a list, we use an KeyIncIterator.
}

func (it *ListIterator) GetValue() values.Value {
	valResult := it.VecIt.Elem().(values.Value)
	it.pos++
	it.VecIt.Next()
	return valResult
}

func (it *ListIterator) GetKeyValuePair() (values.Value, values.Value) {
	keyResult := values.Value{values.INT, it.pos}
	valResult := it.VecIt.Elem().(values.Value)
	it.pos++
	it.VecIt.Next()
	return keyResult, valResult
}

type MapIterator struct { // TODO --- write actual iterator forr Map for this to wrap around.
	KVPairs []values.MapPair
	Len     int
	pos     int
}

func (it *MapIterator) Unfinished() bool {
	return it.pos < it.Len
}

func (it *MapIterator) GetKey() values.Value {
	keyResult := it.KVPairs[it.pos].Key
	it.pos++
	return keyResult
}

func (it *MapIterator) GetValue() values.Value {
	valResult := it.KVPairs[it.pos].Val
	it.pos++
	return valResult
}

func (it *MapIterator) GetKeyValuePair() (values.Value, values.Value) {
	keyResult := it.KVPairs[it.pos].Key
	valResult := it.KVPairs[it.pos].Val
	it.pos++
	return keyResult, valResult
}

type SetIterator struct { // TODO --- write actual iterator for Set for this to wrap around.
	Elements []values.Value
	Len      int
	pos      int
}

func (it *SetIterator) Unfinished() bool {
	return it.pos < it.Len
}

func (it *SetIterator) GetKey() values.Value {
	keyResult := it.Elements[it.pos]
	it.pos++
	return keyResult
}

func (it *SetIterator) GetValue() values.Value {
	valResult := it.Elements[it.pos]
	it.pos++
	return valResult
}

func (it *SetIterator) GetKeyValuePair() (values.Value, values.Value) {
	keyResult := it.Elements[it.pos]
	valResult := it.Elements[it.pos]
	it.pos++
	return keyResult, valResult
}

type StringIterator struct {
	Str string
	pos int
}

func (it *StringIterator) Unfinished() bool {
	return it.pos < len(it.Str)
}

func (it *StringIterator) GetKey() values.Value {
	_, l := utf8.DecodeRuneInString(it.Str[it.pos:])
	keyResult := values.Value{values.INT, it.pos}
	it.pos = it.pos + l
	return keyResult
}

func (it *StringIterator) GetValue() values.Value {
	r, l := utf8.DecodeRuneInString(it.Str[it.pos:])
	valResult := values.Value{values.RUNE, r}
	it.pos = it.pos + l
	return valResult
}

func (it *StringIterator) GetKeyValuePair() (values.Value, values.Value) {
	keyResult := values.Value{values.INT, it.pos}
	r, l := utf8.DecodeRuneInString(it.Str[it.pos:])
	valResult := values.Value{values.RUNE, r}
	it.pos = it.pos + l
	return keyResult, valResult
}

type StructIterator struct {
	Labels []int 
	Values []values.Value
	pos int
}

func (it *StructIterator) Unfinished() bool {
	return it.pos < len(it.Labels)
}

func (it *StructIterator) GetKey() values.Value {
	keyResult := values.Value{values.LABEL, it.Labels[it.pos]}
	it.pos++
	return keyResult
}

func (it *StructIterator) GetValue() values.Value {
	valResult := it.Values[it.pos]
	it.pos++
	return valResult
}

func (it *StructIterator) GetKeyValuePair() (values.Value, values.Value) {
	keyResult := values.Value{values.LABEL, it.Labels[it.pos]}
	valResult := it.Values[it.pos]
	it.pos++
	return keyResult, valResult
}

type TupleIterator struct {
	Elements []values.Value
	Len      int
	pos      int
}

func (it *TupleIterator) Unfinished() bool {
	return it.pos < it.Len
}

func (it *TupleIterator) GetKey() values.Value {
	panic("This doesn't happen.") // If we just want to iterate over the keys of a tuple, we use an KeyIncIterator.
}

func (it *TupleIterator) GetValue() values.Value {
	valResult := it.Elements[it.pos]
	it.pos++
	return valResult
}

func (it *TupleIterator) GetKeyValuePair() (values.Value, values.Value) {
	keyResult := values.Value{values.INT, it.pos}
	valResult := it.Elements[it.pos]
	it.pos++
	return keyResult, valResult
}

