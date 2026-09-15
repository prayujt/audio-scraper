package youtubeimpl

import "testing"

func TestNewCandidateLimit(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{name: "configured value", input: 25, want: 25},
		{name: "zero uses default", input: 0, want: defaultCandidateLimit},
		{name: "negative uses default", input: -1, want: defaultCandidateLimit},
		{name: "maximum value", input: maxCandidateLimit, want: maxCandidateLimit},
		{name: "over maximum is clamped", input: maxCandidateLimit + 1, want: maxCandidateLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.input).candidateLimit; got != tt.want {
				t.Fatalf("candidateLimit = %d, want %d", got, tt.want)
			}
		})
	}
}
