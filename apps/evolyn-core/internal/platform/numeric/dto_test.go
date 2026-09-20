package numeric

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecimalStringJSONContract(t *testing.T) {
	type request struct {
		Quantity  DecimalString `json:"quantity"`
		UnitPrice DecimalString `json:"unitPrice"`
	}

	var got request
	require.NoError(t, json.Unmarshal([]byte(`{"quantity":"003.000","unitPrice":"123.4500"}`), &got))
	require.Equal(t, DecimalString("3"), got.Quantity)
	require.Equal(t, DecimalString("123.45"), got.UnitPrice)

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, `{"quantity":"3","unitPrice":"123.45"}`, string(raw))
}

func TestDecimalStringRejectsNonStringAndNonCanonicalProtocolInputs(t *testing.T) {
	for _, raw := range []string{
		`{"amount":0.1}`,
		`{"amount":null}`,
		`{"amount":" 0.1"}`,
		`{"amount":"1e3"}`,
		`{"amount":"+1"}`,
	} {
		t.Run(raw, func(t *testing.T) {
			var input struct {
				Amount DecimalString `json:"amount"`
			}
			require.Error(t, json.Unmarshal([]byte(raw), &input))
		})
	}
}
