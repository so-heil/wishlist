package validate

import (
	"testing"
)

func TestPassv(t *testing.T) {
	tests := []struct {
		name string
		pass string
		want bool
	}{
		{
			name: "valid",
			pass: "qser56Y0",
			want: true,
		},
		{
			name: "less than 8 chars",
			pass: "TRia19p",
			want: false,
		},
		{
			name: "no number",
			pass: "TGowiqdan",
			want: false,
		},
		{
			name: "no uppercase",
			pass: "askjq1314d",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := passv(tt.pass); got != tt.want {
				t.Errorf("passv() = %v, want %v", got, tt.want)
			}
		})
	}
}
