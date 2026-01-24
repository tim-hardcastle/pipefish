package vm

import (
	"github.com/tim-hardcastle/pipefish/source/values"
	"github.com/wundergraph/astjson"
	"src.elv.sh/pkg/persistent/vector"
)

// For speed and convenience, we hardwire translating JSON into Pipefish values.
func (vm *Vm) jsonToPf(j string, ty values.AbstractType, as bool, tok uint32) values.Value {
	goVal, err := astjson.Parse(j)
	if err != nil {
		return vm.makeError("vm/parse/json", tok, err.Error())
	}
	return vm.goToPf(goVal, ty, as, tok)
}

// Oh hooray, another recursive function for turning Go values into Pipefish! Is this the third or
// the fourth?
//
// It's all fairly straightforward except that we need to special-case the generic `list` and `map`
// types, because we need to pass in the parameterizing type to the recursive `goToPf` call, and
// then having done that we *don't* need to call their validation logic, since that will have
// been taken care of when the elements were converted.
func (vm *Vm) goToPf(goValue *astjson.Value, ty values.AbstractType, as bool, tok uint32) values.Value {
	var canNull bool
	var pfType values.ValueType
	switch ty.Len() {
	case 1 :
		pfType = ty.Types[0]
		canNull = pfType == values.NULL || pfType == values.SUCCESSFUL_VALUE
	case 2 :
		if ty.Types[0] == values.NULL {
			pfType = ty.Types[1]
			canNull = true
		} else {
			return vm.makeError("vm/json/abstract/a", tok)
		}
	default :
		return vm.makeError("vm/json/abstract/b", tok)
	}
	info := vm.ConcreteTypeInfo[pfType]
	cloneInfo, isClone := info.(CloneType)
	switch goValue.Type() {
	case astjson.TypeNull :
		if canNull {
			return values.Value{values.NULL, nil}
		}
		return vm.makeError("vm/json/null", tok)
	case astjson.TypeString :
		if pfType == values.SUCCESSFUL_VALUE {
				pfType = values.STRING
			}
		if pfType == values.STRING || isClone && cloneInfo.Parent == values.STRING {
			return values.Value{pfType, string(goValue.GetStringBytes())}
		}
	case astjson.TypeNumber :
		if i, err := goValue.Int(); err == nil {
			if pfType == values.SUCCESSFUL_VALUE {
				pfType = values.INT
			}
			if pfType == values.INT || isClone && cloneInfo.Parent == values.INT {
				return values.Value{pfType, i}
			}
			return vm.makeError("vm/json/int", tok)
		}
		if i, err := goValue.Float64(); err == nil {
			if pfType == values.SUCCESSFUL_VALUE {
				pfType = values.FLOAT
			}
			if pfType == values.FLOAT || isClone && cloneInfo.Parent == values.FLOAT {
				return values.Value{pfType, i}
			}
			return vm.makeError("vm/json/float", tok)
		}
	case astjson.TypeFalse :
		if pfType == values.BOOL || pfType == values.SUCCESSFUL_VALUE {
			return values.Value{values.BOOL, false}
		}
		return vm.makeError("vm/json/bool/a", tok)
	case astjson.TypeTrue :
		if pfType == values.BOOL || pfType == values.SUCCESSFUL_VALUE {
			return values.Value{values.BOOL, true}
		}
		return vm.makeError("vm/json/bool/b", tok)
	case astjson.TypeArray :
		if pfType == values.SUCCESSFUL_VALUE {
				pfType = values.LIST
			}
		if pfType == values.LIST || isClone && cloneInfo.Parent == values.LIST {
			arr := goValue.GetArray()
			vec := vector.Empty
			insideType := values.MakeAbstractType(values.SUCCESSFUL_VALUE)
			for _, goElement := range arr {
				pfElement := vm.goToPf(goElement, insideType, as, tok)
				if pfElement.T == values.ERROR {
					return pfElement
				}
				vec = vec.Conj(pfElement)
			}
			return values.Value{pfType, vec}
		}
		return vm.makeError("vm/json/list", tok)
	case astjson.TypeObject :
		if pfType == values.SUCCESSFUL_VALUE {
			pfType = values.MAP
		}
		if pfType == values.MAP || isClone && cloneInfo.Parent == values.MAP {
			obj := goValue.GetObject()
			mp := &values.Map{}
			err := values.Value{}
			insideType := values.MakeAbstractType(values.SUCCESSFUL_VALUE)
			obj.Visit(func(k []byte, v *astjson.Value) {
				pfKey := values.Value{values.STRING, string(k)}
				pfValue := vm.goToPf(v, insideType, as, tok)
				if pfValue.T == values.ERROR {
					err = pfValue
					return
				} else {
					mp = mp.Set(pfKey, pfValue)
				}
			})
			if err.T == values.ERROR {
				return err
			}
			return values.Value{pfType, mp}
		}
		if structInfo, ok := info.(StructType); ok {
			obj := goValue.GetObject()
			vals := []values.Value{}
			err := values.Value{}
			fieldNumber := 0
			obj.Visit(func(k []byte, v *astjson.Value) {
				if string(k) != vm.Labels[structInfo.LabelNumbers[fieldNumber]] {
					err = vm.makeError("vm/json/field", tok)
					return
				}
				pfValue := vm.goToPf(v, structInfo.AbstractStructFields[fieldNumber], as, tok)
				if pfValue.T == values.ERROR {
					err = pfValue
				} else {
					vals = append(vals, pfValue)
				}
				fieldNumber++
			})
			if err.T == values.ERROR {
				return err
			}
			return values.Value{pfType, vals}
		}
		return vm.makeError("vm/json/object", tok)
	}
	panic("This should never happen.")
}