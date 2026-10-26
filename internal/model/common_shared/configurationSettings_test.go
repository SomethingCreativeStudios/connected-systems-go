package common_shared

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConstraintPreservesDecodedJSON(t *testing.T) {
	for name, payload := range map[string]string{
		"token values":    `{"type":"AllowedTokens","ref":"parameters/mode","values":["fast"],"reason":"calibration"}`,
		"token pattern":   `{"type":"AllowedTokens","ref":"parameters/mode","pattern":"^[a-z]+$","reason":{"enabled":false,"source":"manual"}}`,
		"numeric values":  `{"type":"AllowedValues","ref":"parameters/count","values":[9007199254740993,1.234567890123456789],"intervals":[[0,9007199254740993]],"significantFigures":18,"reason":"calibration"}`,
		"times":           `{"type":"AllowedTimes","ref":"parameters/time","values":["2026-10-01T00:00:00Z"],"intervals":[["2026-10-01T00:00:00Z","2026-10-02T00:00:00Z"]],"reason":"calibration"}`,
		"inferred values": `{"ref":"parameters/count","values":[1,2],"reason":"internal caller"}`,
	} {
		t.Run(name, func(t *testing.T) {
			var constraint Constraint
			require.NoError(t, json.Unmarshal([]byte(payload), &constraint))
			switch constraint.Type {
			case "AllowedTokens":
				require.NotNil(t, constraint.Tokens)
			case "AllowedValues":
				require.NotNil(t, constraint.Values)
			case "AllowedTimes":
				require.NotNil(t, constraint.Times)
			}
			encoded, err := json.Marshal(constraint)
			require.NoError(t, err)
			requirePreservedJSON(t, payload, encoded)
			constraint.Ref = "parameters/changed"
			encoded, err = json.Marshal(constraint)
			require.NoError(t, err)
			requirePreservedJSON(t, payload, encoded)
		})
	}
}

func TestConstraintConstructedWithoutRaw(t *testing.T) {
	number, figures := 2.5, 3
	for _, tc := range []struct {
		constraint Constraint
		expected   string
	}{
		{
			Constraint{Type: "AllowedTokens", Ref: "parameters/mode", Tokens: &AllowedTokens{Values: []string{"fast"}}},
			`{"type":"AllowedTokens","ref":"parameters/mode","values":["fast"]}`,
		},
		{
			Constraint{Type: "AllowedValues", Ref: "parameters/value", Values: &AllowedValues{
				Values: []ValueItem{{Number: &number}}, Intervals: [][]ValueItem{{{Number: &number}, {Number: &number}}}, SignificantFigures: &figures,
			}},
			`{"type":"AllowedValues","ref":"parameters/value","values":[2.5],"intervals":[[2.5,2.5]],"significantFigures":3}`,
		},
		{
			Constraint{Type: "AllowedTimes", Ref: "parameters/time", Times: &AllowedTimes{
				Values: []string{"2026-10-01T00:00:00Z"}, Intervals: [][]string{{"2026-10-01T00:00:00Z", "2026-10-02T00:00:00Z"}}, SignificantFigures: &figures,
			}},
			`{"type":"AllowedTimes","ref":"parameters/time","values":["2026-10-01T00:00:00Z"],"intervals":[["2026-10-01T00:00:00Z","2026-10-02T00:00:00Z"]],"significantFigures":3}`,
		},
	} {
		t.Run(tc.constraint.Type, func(t *testing.T) {
			encoded, err := json.Marshal(tc.constraint)
			require.NoError(t, err)
			requirePreservedJSON(t, tc.expected, encoded)
		})
	}
}

func TestConstraintDecodeReplacesPreviousVariant(t *testing.T) {
	var constraint Constraint
	require.NoError(t, json.Unmarshal([]byte(`{"type":"AllowedTokens","ref":"parameters/mode","values":["fast"]}`), &constraint))
	require.NotNil(t, constraint.Tokens)
	payload := `{"type":"AllowedValues","ref":"parameters/count","values":[1,2]}`
	require.NoError(t, json.Unmarshal([]byte(payload), &constraint))
	require.Nil(t, constraint.Tokens)
	require.NotNil(t, constraint.Values)
	encoded, err := json.Marshal(constraint)
	require.NoError(t, err)
	requirePreservedJSON(t, payload, encoded)
}
