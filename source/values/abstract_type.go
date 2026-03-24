package values

import "strconv"

func (a AbstractType) String() string {
	result := "["
	sep := ""
	for _, t := range a.Types {
		result = result + sep + strconv.Itoa(int(t))
		sep = ", "
	}
	result = result + "]"
	return result
}

type AbstractType struct {
	Types []ValueType
}

func AbT(args ...ValueType) AbstractType {
	result := AbstractType{[]ValueType{}}
	for _, t := range args {
		result = result.Insert(t)
	}
	return result
}

func (lhs AbstractType) Union(rhs AbstractType) AbstractType {
	i := 0
	j := 0
	result := make([]ValueType, 0, len(lhs.Types)+len(rhs.Types))
	for i < len(lhs.Types) || j < len(rhs.Types) {
		switch {
		case i == len(lhs.Types):
			result = append(result, rhs.Types[j])
			j++
		case j == len(rhs.Types):
			result = append(result, lhs.Types[i])
			i++
		case lhs.Types[i] == rhs.Types[j]:
			result = append(result, lhs.Types[i])
			i++
			j++
		case lhs.Types[i] < rhs.Types[j]:
			result = append(result, lhs.Types[i])
			i++
		case rhs.Types[j] < lhs.Types[i]:
			result = append(result, rhs.Types[j])
			j++
		}
	}
	return AbstractType{result}
}

func (a AbstractType) Insert(v ValueType) AbstractType {
	if len(a.Types) == 0 {
		return AbstractType{[]ValueType{v}}
	}
	for i, t := range a.Types {
		if v == t {
			return a
		}
		if v < t {
			lhs := make([]ValueType, i)
			rhs := make([]ValueType, len(a.Types)-i)
			copy(lhs, a.Types[:i])
			copy(rhs, a.Types[i:])
			lhs = append(lhs, v)
			return AbstractType{append(lhs, rhs...)}
		}
	}
	return AbstractType{append(a.Types, v)}
}

// Because AbstractTypes are ordered we could use binary search for this and there is a threshold beyond which
// it would be quicker, but this must be determined empirically and I haven't done that yet. TODO.
func (a AbstractType) Contains(v ValueType) bool {
	for _, w := range a.Types {
		if v == w {
			return true
		}
	}
	return false
}

func (a AbstractType) Len() int {
	return len(a.Types)
}

func (a AbstractType) Equals(b AbstractType) bool {
	if len(a.Types) != len(b.Types) {
		return false
	}
	for i, v := range a.Types {
		if v != b.Types[i] {
			return false
		}
	}
	return true
}

func (a AbstractType) Is(b ValueType) bool {
	if len(a.Types) != 1 {
		return false
	}
	return a.Types[0] == b
}

func (a AbstractType) IsSubtypeOf(b AbstractType) bool {
	if len(a.Types) > len(b.Types) {
		return false
	}
	i := 0
	for _, t := range a.Types {
		for ; i < len(b.Types) && b.Types[i] < t; i++ {
		}
		if i >= len(b.Types) {
			return false
		}
		if t != b.Types[i] {
			return false
		}
	}
	return true
}

func (a AbstractType) IsProperSubtypeOf(b AbstractType) bool {
	if len(a.Types) > len(b.Types) || len(a.Types) == len(b.Types) {
		return false
	}
	i := 0
	for _, t := range a.Types {
		for ; i < len(b.Types) && b.Types[i] < t; i++ {
		}
		if i >= len(b.Types) {
			return false
		}
		if t != b.Types[i] {
			return false
		}
	}
	return true
}

func (vL AbstractType) Intersect(wL AbstractType) AbstractType {
	result := AbstractType{[]ValueType{}}
	var vix, wix int
	for vix < vL.Len() && wix < wL.Len() {
		if vL.Types[vix] == wL.Types[wix] {
			result.Types = append(result.Types, vL.Types[vix])
			vix++
			wix++
			continue
		}
		if vL.Types[vix] < wL.Types[wix] {
			vix++
			continue
		}
		wix++
	}
	return result
}

func (a AbstractType) PartlyIntersects(b AbstractType) bool {
	intersectionSize := a.Intersect(b).Len()
	return !(a.Len() == intersectionSize || b.Len() == intersectionSize || intersectionSize == 0)
}

func (a AbstractType) Without(b AbstractType) AbstractType {
	rTypes := make([]ValueType, 0, len(a.Types))
	i := 0
	for _, t := range a.Types {
		for ; i < len(b.Types) && b.Types[i] < t; i++ {
		}
		if i >= len(b.Types) {
			rTypes = append(rTypes, t)
			continue
		}
		if t != b.Types[i] {
			rTypes = append(rTypes, t)
		}
	}
	return AbstractType{rTypes}
}

type AbstractTypeInfo struct {
	Name string
	Path string
	AT   AbstractType
	IsMI bool
}

func (aT AbstractTypeInfo) IsMandatoryImport() bool {
	return aT.IsMI
}

// This is the data that goes inside a THUNK value.
type ThunkValue struct {
	MLoc  uint32 // The place in memory where the result of the thunk ends up when you unthunk it.
	CAddr uint32 // The code address to call to unthunk the thunk.
}
