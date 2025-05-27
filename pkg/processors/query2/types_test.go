package query2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryParams_hasInclude(t *testing.T) {
	tests := []struct {
		name        string
		constraints *Constraints
		ok          bool
	}{
		{name: "nil constraints", constraints: nil, ok: false},
		{name: "nil include", constraints: &Constraints{Include: nil}, ok: false},
		{name: "empty include", constraints: &Constraints{Include: []string{}}, ok: false},
		{name: "filled include", constraints: &Constraints{Include: []string{"foo"}}, ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := QueryParams{Constraints: tt.constraints}

			ok := p.hasInclude()

			require.Equal(t, tt.ok, ok)
		})
	}
}
func TestQueryParams_hasKeys(t *testing.T) {
	tests := []struct {
		name        string
		constraints *Constraints
		ok          bool
	}{
		{name: "nil constraints", constraints: nil, ok: false},
		{name: "nil keys", constraints: &Constraints{Keys: nil}, ok: false},
		{name: "empty keys", constraints: &Constraints{Keys: []string{}}, ok: false},
		{name: "filled keys", constraints: &Constraints{Keys: []string{"foo"}}, ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := QueryParams{Constraints: tt.constraints}

			ok := p.hasKeys()

			require.Equal(t, tt.ok, ok)
		})
	}
}
