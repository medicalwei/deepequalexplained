// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Deep equality test via reflection

package deepequalexplained

import (
	"fmt"
	"math"
	"reflect"
	"unsafe"
)

type visit struct {
	a1  unsafe.Pointer
	a2  unsafe.Pointer
	typ reflect.Type
}

func deepValueEqual(v1, v2 reflect.Value, visited map[visit]bool, depth int) error {
	if !v1.IsValid() || !v2.IsValid() {
		if v1.IsValid() == v2.IsValid() {
			return nil
		} else if !v1.IsValid() {
			return fmt.Errorf(" in x is invalid but in y is not")
		} else {
			return fmt.Errorf(" in y is invalid but in x is not")
		}
	}
	if v1.Type() != v2.Type() {
		return fmt.Errorf(" has different types, where in x is %v but in y is %v", v1.Type().Name(), v2.Type().Name())
	}

	hard := func(k reflect.Kind) bool {
		switch k {
		case reflect.Array, reflect.Map, reflect.Slice, reflect.Struct:
			return true
		}
		return false
	}

	if v1.CanAddr() && v2.CanAddr() && hard(v1.Kind()) {
		addr1 := unsafe.Pointer(v1.UnsafeAddr())
		addr2 := unsafe.Pointer(v2.UnsafeAddr())
		if uintptr(addr1) > uintptr(addr2) {
			// Canonicalize order to reduce number of entries in visited.
			// Assumes non-moving garbage collector.
			addr1, addr2 = addr2, addr1
		}

		// Short circuit if references are already seen.
		typ := v1.Type()
		v := visit{addr1, addr2, typ}
		if visited[v] {
			return nil
		}

		// Remember for later.
		visited[v] = true
	}

	switch v1.Kind() {
	case reflect.Array:
		for i := 0; i < v1.Len(); i++ {
			if err := deepValueEqual(v1.Index(i), v2.Index(i), visited, depth+1); err != nil {
				return fmt.Errorf("[%d]%s", i, err.Error())
			}
		}
		return nil
	case reflect.Slice:
		if v1.IsNil() != v2.IsNil() {
			if v1.IsNil() {
				return fmt.Errorf(" in x is nil but in y is not")
			} else {
				return fmt.Errorf(" in y is nil but in x is not")
			}
		}
		if v1.Len() != v2.Len() {
			return fmt.Errorf(" do not have the same length")
		}
		if v1.Pointer() == v2.Pointer() {
			return nil
		}
		for i := 0; i < v1.Len(); i++ {
			if err := deepValueEqual(v1.Index(i), v2.Index(i), visited, depth+1); err != nil {
				return fmt.Errorf("[%d]%s", i, err.Error())
			}
		}
		return nil
	case reflect.Interface:
		if v1.IsNil() || v2.IsNil() {
			if v1.IsNil() == v2.IsNil() {
				return nil
			} else {
				return fmt.Errorf(" do not have the same interface")
			}
		}
		if err := deepValueEqual(v1.Elem(), v2.Elem(), visited, depth+1); err != nil {
			return fmt.Errorf("(Interface)%s", err.Error())
		}
		return nil
	case reflect.Ptr:
		if v1.Pointer() == v2.Pointer() {
			return nil
		}
		if err := deepValueEqual(v1.Elem(), v2.Elem(), visited, depth+1); err != nil {
			return fmt.Errorf("(Ptr)%s", err.Error())
		}
		return nil
	case reflect.Struct:
		for i, n := 0, v1.NumField(); i < n; i++ {
			if err := deepValueEqual(v1.Field(i), v2.Field(i), visited, depth+1); err != nil {
				return fmt.Errorf(".%s%s", v1.Type().Field(i).Name, err.Error())
			}
		}
		return nil
	case reflect.Map:
		if v1.IsNil() != v2.IsNil() {
			if v1.IsNil() {
				return fmt.Errorf(" are not equal, where in x is nil but in y is not")
			} else {
				return fmt.Errorf(" are not equal, where in y is nil but in x is not")
			}
		}
		if v1.Len() != v2.Len() {
			return fmt.Errorf(" do not have the same length, where in x is %d but in y is %d", v1.Len(), v2.Len())
		}
		if v1.Pointer() == v2.Pointer() {
			return nil
		}
		for _, k := range v1.MapKeys() {
			val1 := v1.MapIndex(k)
			val2 := v2.MapIndex(k)
			if !val1.IsValid() {
				return fmt.Errorf("[%v] is invalid in x", k)
			} else if !val2.IsValid() {
				return fmt.Errorf("[%v] is invalid in y", k)
			} else if err := deepValueEqual(v1.MapIndex(k), v2.MapIndex(k), visited, depth+1); err != nil {
				return fmt.Errorf("[%v]%s", k, err.Error())
			}
		}
		return nil
	case reflect.Func:
		if v1.IsNil() && v2.IsNil() {
			return nil
		}
		// Can't do better than this:
		return fmt.Errorf(" has different func")
	default:
		// Trying to compare between two values
		if v1.Kind() == reflect.Float64 && math.IsNaN(v1.Float()) {
			return fmt.Errorf(" in x is NaN float")
		} else if v2.Kind() == reflect.Float64 && math.IsNaN(v2.Float()) {
			return fmt.Errorf(" in y is NaN float")
		} else if fmt.Sprintf("%T", v1) != fmt.Sprintf("%T", v2) {
			return fmt.Errorf(" have different types, where in x is %T but in y is %T", v1, v2)
		} else if fmt.Sprintf("%v", v1) != fmt.Sprintf("%v", v2) {
			return fmt.Errorf(" are not equal, where in x is %v but in y is %v", v1, v2)
		}
		return nil
	}

}

func DeepEqualExplained(x, y interface{}) error {
	if x == nil || y == nil {
		if x == y {
			return nil
		} else if x == nil {
			return fmt.Errorf("x is nil while y is not")
		} else {
			return fmt.Errorf("y is nil while x is not")
		}
	}
	v1 := reflect.ValueOf(x)
	v2 := reflect.ValueOf(y)
	if v1.Type() != v2.Type() {
		return fmt.Errorf("values have different types, where in x is %v but in y is %v", v1.Type().Name(), v2.Type().Name())
	}
	if err := deepValueEqual(v1, v2, make(map[visit]bool), 0); err != nil {
		return fmt.Errorf("values%s", err.Error())
	} else {
		return nil
	}
}
