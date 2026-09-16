package domain

import (
	"strings"
	"testing"
)

func FuzzNewIDs(f *testing.F) {
	f.Add("account-001")
	f.Add("  transaction-001  ")
	f.Add("entry-001")
	f.Add("posting-001")
	f.Add("   ")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		constructors := []struct {
			name string
			new  func(string) (string, error)
		}{
			{
				name: "account ID",
				new: func(value string) (string, error) {
					id, err := NewAccountID(value)
					return id.String(), err
				},
			},
			{
				name: "transaction ID",
				new: func(value string) (string, error) {
					id, err := NewTransactionID(value)
					return id.String(), err
				},
			},
			{
				name: "journal entry ID",
				new: func(value string) (string, error) {
					id, err := NewJournalEntryID(value)
					return id.String(), err
				},
			},
			{
				name: "posting ID",
				new: func(value string) (string, error) {
					id, err := NewPostingID(value)
					return id.String(), err
				},
			},
		}

		cleanInput := strings.TrimSpace(input)
		for _, constructor := range constructors {
			got, err := constructor.new(input)

			if cleanInput == "" {
				if err == nil {
					t.Fatalf("%s accepted an empty value", constructor.name)
				}
				continue
			}

			if err != nil {
				t.Fatalf("%s rejected a non-empty value: %v", constructor.name, err)
			}

			if got != cleanInput {
				t.Fatalf("%s = %q, want trimmed value %q", constructor.name, got, cleanInput)
			}
		}
	})
}
