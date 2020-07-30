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
		wantedWarnCount int
	}{
		{
			"basicDeployment",
			0,
		},
		{
			"configMap",
			0,
		},
		{
			"cronJob",
			0,
		},
		{
			"daemonset",
			0,
		},
		{
			"deployment",
			0,
		},
		{
			"deployment2Containers",
			0,
		},
		{
			"endpoints",
			0,
		},
		{
			"ingress",
			0,
		},
		{
			"job",
			0,
		},
		{
			"namespace",
			0,
		},
		{
			"namespace_w_spec",
			1,
		},
		{
			"networkPolicy",
			0,
		},
		{
			"podDisruptionBudget",
			0,
		},
		{
			"podNodeExporter",
			0,
		},
		{
			"role",
			0,
		},
		{
			"roleBinding",
			0,
		},
		{
			"service",
			0,
		},
		{
			"statefulSet",
			0,
		},
		{
			"issue-48",
			0,
		},
		{
			"runAsUser",
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
