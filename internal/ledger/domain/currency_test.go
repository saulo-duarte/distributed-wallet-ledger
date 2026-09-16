package domain

import (
	"errors"
	"testing"
)

func TestCurrencyValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		code          string
		processedCode string
		wantErr       error
	}{
		{
			name:          "accepts a valid code",
			code:          "BRL",
			processedCode: "BRL",
			wantErr:       nil,
		},
		{
			name:          "accepts a valid code with surrounding whitespace",
			code:          "  BRL  ",
			processedCode: "BRL",
			wantErr:       nil,
		},
		{
			name:          "rejects special characters",
			code:          "BR%",
			processedCode: "",
			wantErr:       ErrInvalidCurrency,
		},
		{
			name:          "rejects numeric characters",
			code:          "BR7",
			processedCode: "",
			wantErr:       ErrInvalidCurrency,
		},
		{
			name:          "rejects codes longer than three characters",
			code:          "BRASIL",
			processedCode: "",
			wantErr:       ErrInvalidCurrency,
		},
		{
			name:          "rejects codes shorter than three characters",
			code:          "BR",
			processedCode: "",
			wantErr:       ErrInvalidCurrency,
		},
		{
			name:          "rejects an empty code",
			code:          "",
			processedCode: "",
			wantErr:       ErrInvalidCurrency,
		},
		{
			name:          "normalizes lowercase codes",
			code:          "brl",
			processedCode: "BRL",
			wantErr:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			currency, err := NewCurrency(tt.code)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewCurrency() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr == nil && currency.String() != tt.processedCode {
				t.Errorf("Currency.String() = %q, want %q", currency.String(), tt.processedCode)
			}
		})
	}

}
