package domain

import (
	"errors"
	"testing"
)

func TestPostingDirection_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		direction PostingDirection
		wantErr   error
	}{
		{
			name:      "debit is valid",
			direction: PostingDirectionDebit,
		},
		{
			name:      "credit is valid",
			direction: PostingDirectionCredit,
		},
		{
			name:      "unknown direction is invalid",
			direction: PostingDirection("unknown"),
			wantErr:   ErrInvalidPostingDirection,
		},
		{
			name:      "empty direction is invalid",
			direction: PostingDirection(""),
			wantErr:   ErrInvalidPostingDirection,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.direction.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestPostingDirection_Opposite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		direction PostingDirection
		want      PostingDirection
	}{
		{
			name:      "debit becomes credit",
			direction: PostingDirectionDebit,
			want:      PostingDirectionCredit,
		},
		{
			name:      "credit becomes debit",
			direction: PostingDirectionCredit,
			want:      PostingDirectionDebit,
		},
		{
			name:      "invalid direction returns an empty value",
			direction: PostingDirection("unknown"),
			want:      PostingDirection(""),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.direction.Opposite(); got != tt.want {
				t.Errorf("Opposite() = %q, want %q", got, tt.want)
			}
		})
	}
}
