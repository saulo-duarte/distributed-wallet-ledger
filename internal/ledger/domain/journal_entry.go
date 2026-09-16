package domain

type JournalEntry struct {
	id            JournalEntryID
	transactionID TransactionID
	currency      Currency
	postings      []Posting
}

func NewJournalEntry(
	id JournalEntryID,
	transactionID TransactionID,
	currency Currency,
	postings []Posting,
) (JournalEntry, error) {

	postingsCopy := make([]Posting, len(postings))
	copy(postingsCopy, postings)

	entry := JournalEntry{
		id:            id,
		transactionID: transactionID,
		currency:      currency,
		postings:      postingsCopy,
	}

	if err := entry.Validate(); err != nil {
		return JournalEntry{}, err
	}

	return entry, nil

}

func (j JournalEntry) Validate() error {
	if j.id.IsZero() {
		return ErrInvalidID
	}

	if j.transactionID.IsZero() {
		return ErrInvalidID
	}

	if err := j.currency.Validate(); err != nil {
		return err
	}

	if len(j.postings) == 0 {
		return ErrJournalEntryWithoutPostings
	}

	hasDebit := false
	hasCredit := false

	for _, p := range j.postings {
		if err := p.Validate(); err != nil {
			return err
		}

		if !p.Amount().Currency().Equal(j.currency) {
			return ErrCurrencyMismatch
		}

		switch p.Direction() {
		case PostingDirectionDebit:
			hasDebit = true
		case PostingDirectionCredit:
			hasCredit = true
		}
	}

	if !hasDebit {
		return ErrJournalEntryWithoutDebit
	}

	if !hasCredit {
		return ErrJournalEntryWithoutCredit
	}

	balanced, err := j.IsBalanced()
	if err != nil {
		return err
	}

	if !balanced {
		return ErrUnbalancedJournalEntry
	}

	return nil
}

func (j JournalEntry) Postings() []Posting {
	postingsCopy := make([]Posting, len(j.postings))
	copy(postingsCopy, j.postings)
	return postingsCopy
}

func (j JournalEntry) TotalDebits() (Money, error) {
	total, err := NewMoney(j.currency, 0)
	if err != nil {
		return Money{}, err
	}

	for _, p := range j.postings {
		if p.Direction() == PostingDirectionDebit {
			total, err = total.Add(p.Amount())
			if err != nil {
				return Money{}, err
			}
		}
	}

	return total, nil
}

func (j JournalEntry) TotalCredits() (Money, error) {
	total, err := NewMoney(j.currency, 0)
	if err != nil {
		return Money{}, err
	}

	for _, p := range j.postings {
		if p.Direction() == PostingDirectionCredit {
			total, err = total.Add(p.Amount())
			if err != nil {
				return Money{}, err
			}
		}
	}

	return total, nil
}

func (j JournalEntry) IsBalanced() (bool, error) {
	debits, err := j.TotalDebits()
	if err != nil {
		return false, err
	}

	credits, err := j.TotalCredits()
	if err != nil {
		return false, err
	}

	return debits.Equal(credits), nil
}

func (j JournalEntry) ID() JournalEntryID {
	return j.id
}

func (j JournalEntry) TransactionID() TransactionID {
	return j.transactionID
}

func (j JournalEntry) Currency() Currency {
	return j.currency
}
