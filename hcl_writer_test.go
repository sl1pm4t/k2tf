package main

import (
	"io/ioutil"
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

var update bool

func init() {
	v := os.Getenv("UPDATE_GOLDEN")
	if strings.ToLower(v) == "true" {
		update = true
	}
}

func testLoadFile(t *testing.T, fileparts ...string) string {
	filename := filepath.Join(fileparts...)
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to load test file, %s: %v", filename, err)
	}

	return string(content)
}

func TestWriteObject(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
	}{
		{
			"basicDeployment",
			"kubernetes_deployment",
		},
		{
			"configMap",
			"kubernetes_config_map",
		},
		{
			"daemonset",
			"kubernetes_daemonset",
		},
		{
			"deployment",
			"kubernetes_deployment",
		},
		{
			"deployment2Containers",
			"kubernetes_deployment",
		},
		{
			"endpoints",
			"kubernetes_endpoints",
		},
		{
			"podNodeExporter",
			"kubernetes_pod",
		},
		{
			"role",
			"kubernetes_role",
		},
		{
			"roleBinding",
			"kubernetes_role_binding",
		},
		{
			"service",
			"kubernetes_service",
		},
		{
			"statefulSet",
			"kubernetes_stateful_set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Generate HCL from test data
			obj := testParseK8SYAML(t, testLoadFile(t, "test-fixtures", tt.name+".yaml"))
			hclFile := hclwrite.NewEmptyFile()
			err := WriteObject(obj, hclFile.Body())
			if err != nil {
				t.Fatal(err)
			}

			// Read our golden file (or optionally write if env var is set)
			goldenFile := filepath.Join("test-fixtures", tt.name+".tf.golden")
			if update {
				ioutil.WriteFile(goldenFile, hclFile.Bytes(), 0644)
			}
			expected := testLoadFile(t, goldenFile)

			// Validate configs are equal
			expectedConfig := parseResourceHCL(t, []byte(expected))
			actualConfig := parseResourceHCL(t, hclFile.Bytes())
			assert.Equal(t,
				expectedConfig,
				actualConfig,
				"resource config should be equal",
			)

			// Validate the generated config is TF schema compliant
			assert.True(
				t,
				validateTerraformConfig(t, tt.resourceType, actualConfig),
				"HCL should pass provider validation",
			)
		})
	}
}

func validateTerraformConfig(t *testing.T, resourceType string, cfg *config.RawConfig) bool {
	// extract our resources rawConfig
	rsrcConfig := terraform.NewResourceConfig(cfg)

	// validate against the Kubernetes provider
	prov := kubernetes.Provider().(*schema.Provider)
	_, errs := prov.ValidateResource(resourceType, rsrcConfig)

	if len(errs) > 0 {
		// log validation errors
		for i, v := range errs {
			t.Fatalf("Validation Error: %d> %v\n", i, v)
		}

		return false
	}

	return true
}

func parseResourceHCL(t *testing.T, hcl []byte) *config.RawConfig {
	// write HCL to temp location where Terraform can load it
	tmpDir, err := ioutil.TempDir("", "ky2tf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write the file
	ioutil.WriteFile(filepath.Join(tmpDir, "hcl.tf"), hcl, os.ModePerm)

	// use terraform to load config from tmp dir
	cfg, err := config.LoadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Resources) == 0 {
		t.Fatal("HCL config load did not return a resource config")
	}

	// extract our resources rawConfig
	return cfg.Resources[0].RawConfig
}

func testParseK8SYAML(t *testing.T, s string) runtime.Object {
	r := strings.NewReader(s)
	objs, err := parseK8SYAML(r)
	if err != nil {
		t.Error("testParseK8SYAML err: ", err)
	}
	return objs[0]
}
