package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/terraform-providers/terraform-provider-kubernetes/kubernetes"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	configMapYAML             string
	basicDeploymentYAML       string
	podNodeExporterYAML       string
	replicationControllerYAML string
	roleYAML                  string
	roleBindingYAML           string
	serviceYAML               string
	statefulSetYAML           string
)

func init() {
	configMapYAML = loadTestFile("configMap.yaml")
	basicDeploymentYAML = loadTestFile("basicDeployment.yaml")
	podNodeExporterYAML = loadTestFile("podNodeExporter.yaml")
	replicationControllerYAML = loadTestFile("replicationController.yml")
	roleYAML = loadTestFile("role.yaml")
	roleBindingYAML = loadTestFile("roleBinding.yaml")
	serviceYAML = loadTestFile("service.yaml")
	statefulSetYAML = loadTestFile("statefulSet.yaml")
}

func loadTestFile(filename string) string {
	pwd, _ := os.Getwd()
	content, err := ioutil.ReadFile(filepath.Join(pwd, "test-fixtures", filename))
	if err != nil {
		log.Fatal(err)
	}

	yaml := string(content)
	return yaml
}

func TestWriteObject(t *testing.T) {
	type args struct {
		yaml         string
		hcl          string
		resourceType string
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
				"kubernetes_deployment",
			},
		},
		{
			"ConfigMap",
			args{
				configMapYAML,
				configMapHCL,
				"kubernetes_config_map",
			},
		},
		{
			"PodVolumesOnly",
			args{
				podVolumesOnlyYAML,
				podVolumesOnlyHCL,
				"kubernetes_pod",
			},
		},
		{
			"PodNodeExporter",
			args{
				podNodeExporterYAML,
				podNodeExporterHCL,
				"kubernetes_pod",
			},
		},
		{
			"role",
			args{
				roleYAML,
				roleHCL,
				"kubernetes_role",
			},
		},
		{
			"roleBinding",
			args{
				roleBindingYAML,
				roleBindingHCL,
				"kubernetes_role_binding",
			},
		},
		{
			"Service",
			args{
				serviceYAML,
				serviceHCL,
				"kubernetes_service",
			},
		},
		{
			"StatefulSet",
			args{
				statefulSetYAML,
				statefulSetHCL,
				"kubernetes_stateful_set",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := mustParseTestYAML(tt.args.yaml)
			f := hclwrite.NewEmptyFile()
			WriteObject(obj, f.Body())

			hclOut := string(f.Bytes())

			// FIXME: flaky test due to ordering of attributes
			assert.Equal(t, tt.args.hcl, hclOut, "HCL should be equal")

			assert.True(t, validateTerraformConfig(t, tt.args.resourceType, f.Bytes()), "HCL should pass provider validation")
		})
	}
}

func validateTerraformConfig(t *testing.T, resourceType string, hcl []byte) bool {
	// write HCL to temp location where Terraform can load it
	tmpDir, err := ioutil.TempDir("", "ky2tf")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	// t.Logf("tmpdir = %s", tmpDir)

	// Write the file
	ioutil.WriteFile(filepath.Join(tmpDir, "hcl.tf"), hcl, os.ModePerm)

	// Invoke terraform to load config
	cfg, err := config.LoadDir(tmpDir)
	if err != nil {
		t.Error(err)
	}

	// extract our resources rawConfig
	rsrcConfig := terraform.NewResourceConfig(cfg.Resources[0].RawConfig)

	// validate against the Kubernetes provider
	prov := kubernetes.Provider().(*schema.Provider)
	_, errs := prov.ValidateResource(resourceType, rsrcConfig)

	if len(errs) > 0 {
		// log validation errors
		for i, v := range errs {
			t.Errorf("Validation Error: %d> %v\n", i, v)
		}

		return false
	}

	os.RemoveAll(tmpDir)
	return true
}

func mustParseTestYAML(s string) runtime.Object {
	r := strings.NewReader(s)
	objs, err := ParseK8SYAML(r)
	if err != nil {
		panic(err)
	}
	return objs[0]
}
