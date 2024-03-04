package money

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMoney_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		v    Money
		want []byte
	}{
		{
			name: "1 positive",
			v:    15000,
			want: []byte("150"),
		},
		{
			name: "2 positive",
			v:    15034,
			want: []byte("150.34"),
		},
		{
			name: "3 positive",
			v:    0,
			want: []byte("0"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := tt.v.MarshalJSON()

			require.Equal(t, got, tt.want)
		})
	}
}

func TestMoney_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want Money
	}{
		{
			name: "1 positive",
			data: []byte("300"),
			want: Money(30000),
		},
		{
			name: "2 positive",
			data: []byte("350.45"),
			want: Money(35045),
		},
		{
			name: "3 positive",
			data: []byte("0"),
			want: Money(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m Money
			_ = m.UnmarshalJSON(tt.data)

			require.Equal(t, m, tt.want)
		})
	}
}
