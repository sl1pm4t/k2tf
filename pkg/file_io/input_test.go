package file_io

import (
	"testing"
)

func Test_readFilesInput(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantObjCount int
	}{
		{
			"../../test-fixtures/service.yaml",
			"../../test-fixtures/service.yaml",
			1,
		},
		{
			"../../test-fixtures",
			"../../test-fixtures",
			25,
		},
		{
			"../../test-fixtures/",
			"../../test-fixtures/",
			25,
		},
		{
			"../../test-fixtures/nested/server-clusterrole.yaml",
			"../../test-fixtures/nested/server-clusterrole.yaml",
			1,
		},
		{
			"../../test-fixtures/nested/",
			"../../test-fixtures/nested/",
			4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readFilesInput(tt.input); len(got) != tt.wantObjCount {
				t.Errorf("readFilesInput() object Count = %d, want %d", len(got), tt.wantObjCount)
			}
		})
	}
}
