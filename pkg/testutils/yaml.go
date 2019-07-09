package testutils

import (
	"github.com/sl1pm4t/k2tf/pkg/k8sparser"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
	"testing"
)

func TestParseYAML(t *testing.T, s string) runtime.Object {
	r := strings.NewReader(s)
	objs, err := k8sparser.ParseYAML(r)
	if err != nil {
		t.Fatalf("test setup error, could not parse test YAML: %v", err)
		return nil
	}
	return objs[0]
}
