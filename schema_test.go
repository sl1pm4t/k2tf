package main

import (
	"testing"
)

func TestSchemaSupportsAttribute(t *testing.T) {
	type args struct {
		attrName string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			"kubernetes_deployment.metadata",
			args{
				"kubernetes_deployment.metadata",
			},
			true,
			false,
		},
		{
			"kubernetes_pod.metadata.labels",
			args{
				"kubernetes_pod.metadata.labels",
			},
			true,
			false,
		},
		{
			"kubernetes_pod.metadata.name",
			args{
				"kubernetes_pod.metadata.name",
			},
			true,
			false,
		},
		{
			"kubernetes_deployment.spec.template.spec.container.name",
			args{
				"kubernetes_deployment.spec.template.spec.container.name",
			},
			true,
			false,
		},
		{
			"kubernetes_deployment.spec.toleration",
			args{
				"kubernetes_deployment.spec.toleration",
			},
			false,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsAttributeSupported(tt.args.attrName)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsAttributeSupported() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsAttributeSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}
