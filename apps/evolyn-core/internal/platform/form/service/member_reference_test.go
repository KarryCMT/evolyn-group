package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectMemberReferencesUsesTableFieldKeys(t *testing.T) {
	items := []any{
		map[string]any{"widget": map[string]any{"widgetName": "owner", "type": "user"}},
		map[string]any{"widget": map[string]any{"widgetName": "reviewers", "type": "usergroup"}},
		map[string]any{"widget": map[string]any{
			"widgetName": "details",
			"type":       "subform",
			"items": []any{
				map[string]any{"widget": map[string]any{"widgetName": "handler", "type": "user"}},
			},
		}},
	}
	values := map[string]any{
		"owner":     "mb_owner",
		"reviewers": []any{"mb_reviewer_a", "mb_reviewer_b"},
		"details": []any{
			map[string]any{"handler": "mb_handler_a"},
			map[string]any{"handler": "mb_handler_b"},
		},
	}

	actual := make(map[string][]string)
	collectMemberReferences(items, values, "", actual)

	require.Equal(t, map[string][]string{
		"owner":                            {"mb_owner"},
		"reviewers":                        {"mb_reviewer_a", "mb_reviewer_b"},
		"__evolyn_subform:details:handler": {"mb_handler_a", "mb_handler_b"},
	}, actual)
}

func TestMemberReferenceValuesSkipsUnsupportedAndBlankValues(t *testing.T) {
	require.Nil(t, memberReferenceValues(12))
	require.Equal(t, []string{"mb_a"}, memberReferenceValues([]any{"", " mb_a ", 1}))
}
