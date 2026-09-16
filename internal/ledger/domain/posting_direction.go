package domain

type PostingDirection string

const (
	PostingDirectionDebit  PostingDirection = "debit"
	PostingDirectionCredit PostingDirection = "credit"
)

func (d PostingDirection) Validate() error {
	switch d {
	case PostingDirectionDebit, PostingDirectionCredit:
		return nil
	default:
		return ErrInvalidPostingDirection
	}
}

func (d PostingDirection) Opposite() PostingDirection {
	switch d {
	case PostingDirectionDebit:
		return PostingDirectionCredit
	case PostingDirectionCredit:
		return PostingDirectionDebit
	default:
		return ""
	}
}
