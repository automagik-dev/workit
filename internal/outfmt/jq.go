package outfmt

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// ApplyJQ applies a jq expression to JSON bytes and returns the result.
// The result is raw jq output -- NOT re-wrapped in a JSON envelope.
func ApplyJQ(jsonBytes []byte, expression string) ([]byte, error) {
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("invalid jq expression: %s — %w", expression, err)
	}

	var input any
	if err := json.Unmarshal(jsonBytes, &input); err != nil {
		return nil, fmt.Errorf("parse JSON for jq: %w", err)
	}

	iter := query.Run(input)
	var results []byte

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return nil, fmt.Errorf("jq error: %w", err)
		}

		b, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("marshal jq result: %w", err)
		}
		if len(results) > 0 {
			results = append(results, '\n')
		}
		results = append(results, b...)
	}

	return results, nil
}
