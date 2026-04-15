package infra

import (
	"iter"
	"reflect"
)

func IterStruct(foo any) iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		v := reflect.ValueOf(foo)
		typ := v.Type()
		if typ.Kind() == reflect.Pointer {
			v = v.Elem()
			typ = v.Type()
		}

		for idx := range v.NumField() {
			val := v.Field(idx)
			if val.Type().Kind() == reflect.Pointer {
				val = val.Elem()
			}
			if !yield(typ.Field(idx).Name, val.Interface()) {
				return
			}
		}
	}
}
