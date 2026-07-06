package mappers

import (
	"testing"
)

func TestMapProviderStatus(t *testing.T) {
	tests := []struct {
		name           string
		providerStatus string
		want           string
	}{
		{
			name:           "should map APPROVED to EXECUTED",
			providerStatus: "APPROVED",
			want:           "EXECUTED",
		},
		{
			name:           "should map any other value to REJECTED",
			providerStatus: "PENDING",
			want:           "REJECTED",
		},
		{
			name:           "should map empty string to REJECTED",
			providerStatus: "",
			want:           "REJECTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapProviderStatus(tt.providerStatus); got != tt.want {
				t.Errorf("MapProviderStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
