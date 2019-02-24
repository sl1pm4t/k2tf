package main

import (
	"testing"
)

func TestSchemaSupportsAttribute(t *testing.T) {
	type args struct {
		resName  string
		attrName string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			"kubernetes_pod.metadata.name",
			args{
				"kubernetes_pod",
				"metadata.name",
			},
			true,
			false,
		},
		{
			"kubernetes_deployment.spec.template.spec.container.name",
			args{
				"kubernetes_deployment",
				"spec.template.spec.container.name",
			},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SchemaSupportsAttribute(tt.args.resName, tt.args.attrName)
			if (err != nil) != tt.wantErr {
				t.Errorf("SchemaSupportsAttribute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SchemaSupportsAttribute() = %v, want %v", got, tt.want)
			}
		})
	}
}
