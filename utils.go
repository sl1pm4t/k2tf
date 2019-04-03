package main

import (
	"fmt"
	"reflect"
	"strings"
)

// IsZero reports whether is considered the zero / empty / unset value fo the type
func IsZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice:
		return v.IsNil() || v.Len() == 0
	case reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

// IsInlineStruct looks at the json tag of the given StructField, to determine
// if it has been marked as "inline"
// e.g. someField string `json:",inline"`
func IsInlineStruct(field *reflect.StructField) bool {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		fmt.Printf("WARNING - field [%s] has no json tag value", field.Name)
		return false
	}

	comma := strings.Index(jsonTag, ",")
	if comma == -1 {
		return false
	}

	tagParts := strings.Split(jsonTag, ",")
	for _, part := range tagParts {
		if part == "inline" {
			return true
		}
	}

	return false
}
