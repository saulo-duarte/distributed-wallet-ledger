package dynamo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"financial-ledger/internal/ledger/application/wallet"
	"financial-ledger/internal/ledger/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type walletBalanceItem struct {
	PK                         string `dynamodbav:"pk"`
	SK                         string `dynamodbav:"sk"`
	CreatedAt                  string `dynamodbav:"created_at"`
	WalletID                   string `dynamodbav:"wallet_id"`
	Currency                   string `dynamodbav:"currency"`
	LedgerBalanceMinorUnits    int64  `dynamodbav:"ledger_balance_minor_units"`
	AvailableBalanceMinorUnits int64  `dynamodbav:"available_balance_minor_units"`
	UpdatedAt                  string `dynamodbav:"updated_at"`
}

type WalletBalanceProjectionRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewWalletBalanceProjectionRepository(
	client *dynamodb.Client,
	tableName string,
) *WalletBalanceProjectionRepository {
	return &WalletBalanceProjectionRepository{
		client:    client,
		tableName: tableName,
	}
}

var _ wallet.WalletBalanceProjectionRepository = (*WalletBalanceProjectionRepository)(nil)

func (r *WalletBalanceProjectionRepository) SaveWalletBalance(
	ctx context.Context,
	balance wallet.WalletBalance,
) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	pk := fmt.Sprintf("WALLET#%s", balance.WalletID.String())
	sk := "BALANCE"

	item := walletBalanceItem{
		PK:                         pk,
		SK:                         sk,
		CreatedAt:                  now,
		WalletID:                   balance.WalletID.String(),
		Currency:                   balance.Currency.String(),
		LedgerBalanceMinorUnits:    balance.LedgerBalanceMinorUnits,
		AvailableBalanceMinorUnits: balance.AvailableBalanceMinorUnits,
		UpdatedAt:                  now,
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("marshal wallet balance item: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(pk) OR updated_at <= :new_updated_at"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":new_updated_at": &types.AttributeValueMemberS{Value: now},
		},
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return nil
		}
		return fmt.Errorf("put wallet balance item in dynamodb: %w", err)
	}

	return nil
}

func (r *WalletBalanceProjectionRepository) GetWalletBalance(
	ctx context.Context,
	walletID domain.WalletID,
) (wallet.WalletBalance, error) {
	pk := fmt.Sprintf("WALLET#%s", walletID.String())
	sk := "BALANCE"

	output, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: pk},
			"sk": &types.AttributeValueMemberS{Value: sk},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return wallet.WalletBalance{}, fmt.Errorf("get wallet balance from dynamodb: %w", err)
	}

	if output.Item == nil {
		return wallet.WalletBalance{}, wallet.ErrWalletNotFound
	}

	var item walletBalanceItem
	if err := attributevalue.UnmarshalMap(output.Item, &item); err != nil {
		return wallet.WalletBalance{}, fmt.Errorf("unmarshal wallet balance item: %w", err)
	}

	curr, err := domain.NewCurrency(item.Currency)
	if err != nil {
		return wallet.WalletBalance{}, fmt.Errorf("parse currency %q: %w", item.Currency, err)
	}

	wID, err := domain.NewWalletID(item.WalletID)
	if err != nil {
		return wallet.WalletBalance{}, fmt.Errorf("parse wallet id %q: %w", item.WalletID, err)
	}

	return wallet.WalletBalance{
		WalletID:                   wID,
		Currency:                   curr,
		LedgerBalanceMinorUnits:    item.LedgerBalanceMinorUnits,
		AvailableBalanceMinorUnits: item.AvailableBalanceMinorUnits,
	}, nil
}
