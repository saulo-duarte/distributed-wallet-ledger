package domain

import "strings"

type Currency struct {
	code string
}

func NewCurrency(code string) (Currency, error) {
	cleanCode := strings.ToUpper(strings.TrimSpace(code))

	curr := Currency{code: cleanCode}
	if err := curr.Validate(); err != nil {
		return Currency{}, err
	}

	return curr, nil
}

func (c Currency) String() string {
	return c.code
}

func (c Currency) Equal(other Currency) bool {
	return c.code == other.code
}

func (c Currency) Validate() error {
	if len(c.code) != 3 {
		return ErrInvalidCurrency
	}

	for _, r := range c.code {
		if r < 'A' || r > 'Z' {
			return ErrInvalidCurrency
		}
	}

	return nil
}
