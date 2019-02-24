package main

import (
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
)

func init() {

}

func TestIsInlineStruct(t *testing.T) {
	type args struct {
		field      string
		parentType reflect.Type
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"VolumeSource",
			args{
				"VolumeSource",
				reflect.TypeOf(v1.Volume{}),
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sfield, ok := tt.args.parentType.FieldByName(tt.args.field)
			if !ok {
				t.Errorf("Field %s not found", tt.args.field)
			}
			if got := IsInlineStruct(&sfield); got != tt.want {
				t.Errorf("IsInlineStruct() = %v, want %v", got, tt.want)
			}
		})
	}
}
