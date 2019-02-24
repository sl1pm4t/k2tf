package main

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestWriteObject(t *testing.T) {
	type args struct {
		yaml string
		hcl  string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"BasicDeployment",
			args{
				basicDeploymentYAML,
				basicDeploymentHCL,
			},
		},
		{
			"ConfigMap",
			args{
				configMapYAML,
				configMapHCL,
			},
		},
		{
			"PodVolumes",
			args{
				podVolumesOnlyYAML,
				podVolumesOnlyHCL,
			},
		},
	}
	// dmp := diffmatchpatch.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := mustParseTestYAML(tt.args.yaml)
			f := hclwrite.NewEmptyFile()
			WriteObject(obj, f.Body())

			hclOut := string(f.Bytes())

			assert.Equal(t, tt.args.hcl, hclOut, "HCL should be equal")

			// diffs := dmp.DiffMain(tt.args.hcl, hclOut, false)

			// if len(diffs) > 0 {
			// 	t.Errorf("HCL did not match (%d differences): \n%s", len(diffs), dmp.DiffPrettyText(diffs))
			// }
		})
	}
}

func mustParseTestYAML(s string) runtime.Object {
	r := strings.NewReader(s)
	objs, err := ParseK8SYAML(r)
	if err != nil {
		panic(err)
	}
	return objs[0]
}
