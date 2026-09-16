package domain

type Posting struct {
	id        PostingID
	accountID AccountID
	direction PostingDirection
	amount    Money
}

func NewPosting(
	id PostingID,
	accountID AccountID,
	direction PostingDirection,
	amount Money,
) (Posting, error) {
	posting := Posting{
		id:        id,
		accountID: accountID,
		direction: direction,
		amount:    amount,
	}

	if err := posting.Validate(); err != nil {
		return Posting{}, err
	}

	return posting, nil
}

func (p Posting) ID() PostingID {
	return p.id
}

func (p Posting) AccountID() AccountID {
	return p.accountID
}

func (p Posting) Direction() PostingDirection {
	return p.direction
}

func (p Posting) Amount() Money {
	return p.amount
}

func (p Posting) Validate() error {
	if p.id.IsZero() {
		return ErrInvalidID
	}

	if p.accountID.IsZero() {
		return ErrInvalidID
	}

	if err := p.direction.Validate(); err != nil {
		return err
	}

	if err := p.amount.Validate(); err != nil {
		return err
	}

	if p.amount.IsZero() {
		return ErrAmountMustBePositive
	}

	return nil
}
