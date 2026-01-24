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
	result := vm.makeError("vm/json/convert", tok)
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
			result = values.Value{pfType, string(goValue.GetStringBytes())}
		}
	case astjson.TypeNumber :
		i, err := goValue.Int()
		if pfType == values.SUCCESSFUL_VALUE {
			if err == nil {
				pfType = values.INT
			} else {
				pfType = values.FLOAT
			}
		}
		if err == nil && (pfType == values.INT || isClone && cloneInfo.Parent == values.INT) {
			result = values.Value{pfType, i}
		}
		if pfType == values.FLOAT || isClone && cloneInfo.Parent == values.FLOAT {
			result = values.Value{pfType, goValue.GetFloat64()}
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
			if isClone && len(cloneInfo.TypeArguments) == 1 && cloneInfo.TypeArguments[0].T == values.TYPE {
				insideType = cloneInfo.TypeArguments[0].V.(values.AbstractType)
				if !as {
					pfType = values.LIST
				}
			}
			for _, goElement := range arr {
				pfElement := vm.goToPf(goElement, insideType, as, tok)
				if pfElement.T == values.ERROR {
					return pfElement
				}
				vec = vec.Conj(pfElement)
			}
			result = values.Value{pfType, vec}
		}
	case astjson.TypeObject :
		if pfType == values.SUCCESSFUL_VALUE {
			pfType = values.MAP
		}
		if pfType == values.MAP || isClone && cloneInfo.Parent == values.MAP {
			obj := goValue.GetObject()
			mp := &values.Map{}
			err := values.Value{}
			insideType := values.MakeAbstractType(values.SUCCESSFUL_VALUE)
			stringlikeType := values.STRING
			if isClone && len(cloneInfo.TypeArguments) == 2 && 
			cloneInfo.TypeArguments[0].T == values.TYPE && cloneInfo.TypeArguments[1].T == values.TYPE {
				keyType := cloneInfo.TypeArguments[0].V.(values.AbstractType)
				if keyType.Len() != 1 {
					return vm.makeError("vm/json/key/concrete", tok)
				}
				stringlikeType = keyType.Types[0]
				keyInfo := vm.ConcreteTypeInfo[stringlikeType]
				if !(stringlikeType == values.STRING || keyInfo.IsClone() && keyInfo.(CloneType).Parent == values.STRING) {
					return vm.makeError("vm/json/key/string", tok)
				}
				insideType = cloneInfo.TypeArguments[1].V.(values.AbstractType)
				if !as {
					pfType = values.MAP
				}
			}
			obj.Visit(func(k []byte, v *astjson.Value) {
				pfKey := values.Value{stringlikeType, string(k)}
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
			result = values.Value{pfType, mp}
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
			result = values.Value{pfType, vals}
		}
	}
	return result
}