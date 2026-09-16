package domain

import "testing"

func FuzzNewCurrency(f *testing.F) {
	f.Add("BRL")
	f.Add(" brl ")
	f.Add("USD")
	f.Add("")
	f.Add("BR7")

	f.Fuzz(func(t *testing.T, input string) {
		currency, err := NewCurrency(input)
		if err != nil {
			return
		}

		code := currency.String()
		if len(code) != 3 {
			t.Fatalf("valid currency code has length %d, want 3", len(code))
		}

		for _, character := range code {
			if character < 'A' || character > 'Z' {
				t.Fatalf("valid currency code contains non-uppercase ASCII character: %q", code)
			}
		}

		if err := currency.Validate(); err != nil {
			t.Fatalf("currency returned without error but failed validation: %v", err)
		}
	})
}
