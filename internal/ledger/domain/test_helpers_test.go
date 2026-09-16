package domain

func fakeValidAccountID() AccountID {
	return AccountID("account-001")
}

func fakeValidPostingID() PostingID {
	return PostingID("postingID")
}

func fakeZeroID() AccountID {
	return AccountID("")
}

func fakeValidCurrency(code string) Currency {
	currency, err := NewCurrency(code)
	if err != nil {
		panic(err)
	}

	return currency
}

func fakeCurrency(code string) Currency {
	return Currency{code: code}
}
