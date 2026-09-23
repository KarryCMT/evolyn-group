package label

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// labelContractVector 由 Go 与 TypeScript 共读，防止两端 LabelSchema 校验语义漂移。
type labelContractVector struct {
	Name           string          `json:"name"`
	Schema         json.RawMessage `json:"schema"`
	ExpectedIssues []Issue         `json:"expectedIssues"`
}

func loadLabelContractVectors(t *testing.T) []labelContractVector {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", "docs", "contracts", "label-schema-test-vectors.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "标签协议向量文件应位于 docs/contracts/")
	var doc struct {
		Vectors []labelContractVector `json:"vectors"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.Vectors)
	return doc.Vectors
}

func TestLabelSchemaContractVectors(t *testing.T) {
	for _, vector := range loadLabelContractVectors(t) {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			var schema Schema
			require.NoError(t, json.Unmarshal(vector.Schema, &schema))
			require.Equal(t, vector.ExpectedIssues, Validate(&schema))
		})
	}
}
