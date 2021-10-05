package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sl1pm4t/k2tf/pkg/testutils"

	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/stretchr/testify/assert"
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
			"podDisruptionBudget",
			"kubernetes_pod_disruption_budget",
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
		{
			"issue-48",
			"kubernetes_replication_controller",
			0,
		},
		{
			"certificateSigningRequest",
			"kubernetes_certificate_signing_request",
			0,
		},
		{
			"clusterRole",
			"kubernetes_cluster_role",
			0,
		},
		{
			"issue-28",
			"kubernetes_daemonset",
			0,
		},
		{
			"storageClass",
			"kubernetes_storage_class",
			0,
		},
		{
			"replicationController",
			"kubernetes_replication_controller",
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
			assert.Equal(t, expected, string(hclFile.Bytes()), "should be equal")

			// Validate warning count
			assert.Equal(t, tt.wantedWarnCount, warnCount, "conversion warning count should match")
		})
	}
}
