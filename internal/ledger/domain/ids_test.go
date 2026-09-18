package domain

import "testing"

func TestIDConstructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		newID func(string) (string, error)
		input string
		want  string
	}{
		{
			name: "creates an account ID",
			newID: func(value string) (string, error) {
				id, err := NewAccountID(value)
				return id.String(), err
			},
			input: "  account-001  ",
			want:  "account-001",
		},
		{
			name: "creates a transaction ID",
			newID: func(value string) (string, error) {
				id, err := NewTransactionID(value)
				return id.String(), err
			},
			input: " transaction-001 ",
			want:  "transaction-001",
		},
		{
			name: "creates a journal entry ID",
			newID: func(value string) (string, error) {
				id, err := NewJournalEntryID(value)
				return id.String(), err
			},
			input: " entry-001 ",
			want:  "entry-001",
		},
		{
			name: "creates a posting ID",
			newID: func(value string) (string, error) {
				id, err := NewPostingID(value)
				return id.String(), err
			},
			input: " posting-001 ",
			want:  "posting-001",
		},
		{
			name: "creates a hold ID",
			newID: func(value string) (string, error) {
				id, err := NewHoldID(value)
				return id.String(), err
			},
			input: " hold-001 ",
			want:  "hold-001",
		},
		{
			name: "creates an event ID",
			newID: func(value string) (string, error) {
				id, err := NewEventID(value)
				return id.String(), err
			},
			input: " event-001 ",
			want:  "event-001",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.newID(tt.input)
			if err != nil {
				t.Fatalf("constructor returned an unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIDConstructorsRejectEmptyValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		newID func(string) (string, error)
	}{
		{
			name: "account ID",
			newID: func(value string) (string, error) {
				id, err := NewAccountID(value)
				return id.String(), err
			},
		},
		{
			name: "transaction ID",
			newID: func(value string) (string, error) {
				id, err := NewTransactionID(value)
				return id.String(), err
			},
		},
		{
			name: "journal entry ID",
			newID: func(value string) (string, error) {
				id, err := NewJournalEntryID(value)
				return id.String(), err
			},
		},
		{
			name: "posting ID",
			newID: func(value string) (string, error) {
				id, err := NewPostingID(value)
				return id.String(), err
			},
		},
		{
			name: "hold ID",
			newID: func(value string) (string, error) {
				id, err := NewHoldID(value)
				return id.String(), err
			},
		},
		{
			name: "event ID",
			newID: func(value string) (string, error) {
				id, err := NewEventID(value)
				return id.String(), err
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.newID("   ")
			if err == nil {
				t.Fatal("expected an error for an empty ID, got nil")
			}
		})
	}
}

func TestIDZeroValues(t *testing.T) {
	t.Parallel()

	if !AccountID("").IsZero() {
		t.Error("empty AccountID should be zero")
	}
	if !TransactionID("").IsZero() {
		t.Error("empty TransactionID should be zero")
	}
	if !JournalEntryID("").IsZero() {
		t.Error("empty JournalEntryID should be zero")
	}
	if !PostingID("").IsZero() {
		t.Error("empty PostingID should be zero")
	}
	if !HoldID("").IsZero() {
		t.Error("empty HoldID should be zero")
	}
	if !EventID("").IsZero() {
		t.Error("empty EventID should be zero")
	}

	if AccountID("account-001").IsZero() {
		t.Error("non-empty AccountID should not be zero")
	}
	if TransactionID("transaction-001").IsZero() {
		t.Error("non-empty TransactionID should not be zero")
	}
	if JournalEntryID("entry-001").IsZero() {
		t.Error("non-empty JournalEntryID should not be zero")
	}
	if PostingID("posting-001").IsZero() {
		t.Error("non-empty PostingID should not be zero")
	}
	if HoldID("hold-001").IsZero() {
		t.Error("non-empty HoldID should not be zero")
	}
	if EventID("event-001").IsZero() {
		t.Error("non-empty EventID should not be zero")
	}
}
