package cmd

import "testing"

func TestCompleteWordsStopsAfterTerminator(t *testing.T) {
	cases := []struct {
		name  string
		cword int
		words []string
	}{
		{
			name:  "current-is-terminator",
			cword: 1,
			words: []string{"wk", "--"},
		},
		{
			name:  "after-terminator",
			cword: 3,
			words: []string{"wk", "auth", "--", "-"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := completeWords(tc.cword, tc.words)
			if err != nil {
				t.Fatalf("completeWords: %v", err)
			}
			if len(got) != 0 {
				t.Fatalf("expected no suggestions, got %v", got)
			}
		})
	}
}
