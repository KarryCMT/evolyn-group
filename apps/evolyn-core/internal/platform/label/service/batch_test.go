package service

import (
	"testing"

	labelapp "evolyn/internal/platform/label"

	"github.com/stretchr/testify/require"
)

func TestNormalizeBatchRecordIDsRejectsInvalidAndDuplicateValues(t *testing.T) {
	ids, err := normalizeBatchRecordIDs([]string{" 1 ", "2", "3"})
	require.NoError(t, err)
	require.Equal(t, []uint{1, 2, 3}, ids)

	_, err = normalizeBatchRecordIDs([]string{"1", "1"})
	require.ErrorIs(t, err, labelapp.ErrBatchInvalid)
	_, err = normalizeBatchRecordIDs([]string{"not-an-id"})
	require.ErrorIs(t, err, labelapp.ErrBatchInvalid)
}

func TestRenderTaskCodeValidation(t *testing.T) {
	require.True(t, validRenderTaskCode("lrt_0123456789abcdef01234567"))
	require.False(t, validRenderTaskCode("label_0123456789abcdef"))
}
