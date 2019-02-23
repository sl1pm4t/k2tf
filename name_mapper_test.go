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
			"replicas",
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
			"container",
			args{
				reflect.StructField{
					Name: "Container",
					Tag:  `json:"containers" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=containers"`,
				},
			},
			"container",
		},
		{
			"port",
			args{
				reflect.StructField{
					Name: "ContainerPort",
					Tag:  `json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"containerPort" protobuf:"bytes,6,rep,name=ports"`,
				},
			},
			"port",
		},
		{
			"match_labels",
			args{
				reflect.StructField{
					Name: "MatchLabels",
					Tag:  `json:"matchLabels,omitempty" protobuf:"bytes,1,rep,name=matchLabels"`,
				},
			},
			"match_labels",
		},
		{
			"volume_source",
			args{
				reflect.StructField{
					Name: "VolumeSource",
					Tag:  `json:",inline" protobuf:"bytes,2,opt,name=volumeSource"`,
				},
			},
			"volume_source",
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

func Test_normalizeTerraformName(t *testing.T) {
	type args struct {
		s          string
		toSingular bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"labels",
			args{
				"labels",
				true,
			},
			"labels",
		},
		{
			"match_labels",
			args{
				"matchLabels",
				true,
			},
			"match_labels",
		},
		{
			"metadata",
			args{
				"metadata",
				true,
			},
			"metadata",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTerraformName(tt.args.s, tt.args.toSingular); got != tt.want {
				t.Errorf("normalizeTerraformName() = %v, want %v", got, tt.want)
			}
		})
	}
}
