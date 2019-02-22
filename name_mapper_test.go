package main

import (
	"reflect"
	"testing"
)

func TestToTerraformAttributeName(t *testing.T) {
	type args struct {
		field reflect.StructField
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Replicas",
			args{
				reflect.StructField{
					Name: "Replicas",
					Tag:  `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`,
				},
			},
			"replicas",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToTerraformAttributeName(tt.args.field); got != tt.want {
				t.Errorf("ToTerraformAttributeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToTerraformSubBlockName(t *testing.T) {
	type args struct {
		field reflect.StructField
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Container",
			args{
				reflect.StructField{
					Name: "Container",
					Tag:  `json:"containers" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=containers"`,
				},
			},
			"container",
		},
		{
			"ContainerPort",
			args{
				reflect.StructField{
					Name: "ContainerPort",
					Tag:  `json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"containerPort" protobuf:"bytes,6,rep,name=ports"`,
				},
			},
			"port",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToTerraformSubBlockName(tt.args.field); got != tt.want {
				t.Errorf("ToTerraformSubBlockName() = %v, want %v", got, tt.want)
			}
		})
	}
}
