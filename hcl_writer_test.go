package main

import (
	"github.com/sl1pm4t/k2tf/pkg/testutils"
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
		name            string
		resourceType    string
		wantedWarnCount int
	}{
		{
			"basicDeployment",
			"kubernetes_deployment",
			0,
		},
		{
			"configMap",
			"kubernetes_config_map",
			0,
		},
		{
			"cronJob",
			"kubernetes_cron_job",
			0,
		},
		{
			"daemonset",
			"kubernetes_daemonset",
			0,
		},
		{
			"deployment",
			"kubernetes_deployment",
			0,
		},
		{
			"deployment2Containers",
			"kubernetes_deployment",
			0,
		},
		{
			"endpoints",
			"kubernetes_endpoints",
			0,
		},
		{
			"ingress",
			"kubernetes_ingress",
			0,
		},
		{
			"job",
			"kubernetes_job",
			0,
		},
		{
			"namespace",
			"kubernetes_namespace",
			0,
		},
		{
			"namespace_w_spec",
			"kubernetes_namespace",
			1,
		},
		{
			"networkPolicy",
			"kubernetes_network_policy",
			0,
		},
		{
			"podNodeExporter",
			"kubernetes_pod",
			0,
		},
		{
			"role",
			"kubernetes_role",
			0,
		},
		{
			"roleBinding",
			"kubernetes_role_binding",
			0,
		},
		{
			"service",
			"kubernetes_service",
			0,
		},
		{
			"statefulSet",
			"kubernetes_stateful_set",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Generate HCL from test data
			obj := testutils.TestParseYAML(t, testLoadFile(t, "test-fixtures", tt.name+".yaml"))
			hclFile := hclwrite.NewEmptyFile()
			warnCount, err := WriteObject(obj, hclFile.Body())
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

			// Validate warning count
			assert.Equal(t, tt.wantedWarnCount, warnCount, "conversion warning count should match")
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
	tmpDir, err := ioutil.TempDir("", "k2tf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write the file
	err = ioutil.WriteFile(filepath.Join(tmpDir, "hcl.tf"), hcl, os.ModePerm)
	if err != nil {
		t.Fatalf("test setup error: %v", err)
	}

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
