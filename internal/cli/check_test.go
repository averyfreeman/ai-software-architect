package cli

import (
	"testing"

	"github.com/averyfreeman/adr-repo-governance/internal/adr"
)

func TestIsReviewableStatus(t *testing.T) {
	tests := []struct {
		name   string
		status adr.Status
		want   bool
	}{
		{name: "proposed", status: adr.StatusProposed, want: true},
		{name: "adopted", status: adr.StatusAdopted, want: true},
		{name: "rejected", status: adr.StatusRejected, want: false},
		{name: "deprecated", status: adr.StatusDeprecated, want: false},
		{name: "superseded", status: adr.StatusSuperseded, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReviewableStatus(tt.status); got != tt.want {
				t.Errorf("isReviewableStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
