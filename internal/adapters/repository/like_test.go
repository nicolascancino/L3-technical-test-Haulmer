package repository

import "testing"

func TestEscapeLike(t *testing.T) {
	tests := map[string]struct{ in, want string }{
		"plain":      {"Ada", "Ada"},
		"percent":    {"a%b", `a\%b`},
		"underscore": {"a_b", `a\_b`},
		"backslash":  {`a\b`, `a\\b`},
		"all":        {`%_\`, `\%\_\\`},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := escapeLike(tt.in); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
