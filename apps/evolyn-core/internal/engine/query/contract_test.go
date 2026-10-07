package query

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

type contractVector struct {
	Name               string          `json:"name"`
	Query              json.RawMessage `json:"query"`
	ExpectedIssues     []Issue         `json:"expectedIssues"`
	ExpectedNormalized json.RawMessage `json:"expectedNormalized"`
}

func loadContract(t *testing.T) (FieldCatalog, []contractVector) {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", "docs", "contracts", "dashboard-query-test-vectors.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var document struct {
		Fields  FieldCatalog     `json:"fields"`
		Vectors []contractVector `json:"vectors"`
	}
	require.NoError(t, json.Unmarshal(raw, &document))
	require.NotEmpty(t, document.Vectors)
	return document.Fields, document.Vectors
}

func TestQueryContractVectors(t *testing.T) {
	fields, vectors := loadContract(t)
	for _, vector := range vectors {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			result := NormalizeJSON(vector.Query, fields, DefaultBudget())
			actualIssues := make([]Issue, len(result.Issues))
			for i := range result.Issues {
				actualIssues[i] = Issue{Path: result.Issues[i].Path, Code: result.Issues[i].Code}
			}
			require.Equal(t, vector.ExpectedIssues, actualIssues)
			if len(vector.ExpectedNormalized) == 0 {
				require.Nil(t, result.Document)
				return
			}
			normalized, err := json.Marshal(result.Document)
			require.NoError(t, err)
			require.JSONEq(t, string(vector.ExpectedNormalized), string(normalized))
			require.NotNil(t, result.Plan)
		})
	}
}

func TestQueryPlanMeasuresNestedFilter(t *testing.T) {
	document := Document{
		Version: Version,
		Filter: &Expression{Type: "group", Conjunction: "and", Children: []Expression{
			{Type: "condition", Field: "region", Operator: OperatorEQ, Value: "华东"},
			{Type: "group", Conjunction: "or", Children: []Expression{
				{Type: "condition", Field: "amount", Operator: OperatorGT, Value: 10.0},
			}},
		}},
		Sorts: []Sort{}, Paging: Paging{Page: 1, PageSize: 20},
	}
	result := Validate(document, nil, DefaultBudget())
	require.Empty(t, result.Issues)
	require.Equal(t, Complexity{Depth: 3, Conditions: 2}, result.Plan.Complexity)
}
