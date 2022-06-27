package hyphe

import (
	"context"
	"encoding/json"
	"fmt"
)

type Side string
type OrderType string
type OrderStatus string

const ORDER_ENDPOINT = "/orders"

const (
	Side_SELL Side = "sell"
	Side_BUY  Side = "buy"

	OrderType_QUOTE  OrderType = "quote"
	OrderType_MARKET OrderType = "market"

	OrderStatus_NEW              OrderStatus = "new"
	OrderStatus_PARTIALLY_FILLED OrderStatus = "partially_filled"
	OrderStatus_FILLED           OrderStatus = "filled"
	OrderStatus_CANCELED         OrderStatus = "canceled"
	OrderStatus_EXPIRED          OrderStatus = "expired"
	ORderStatus_REJECTED         OrderStatus = "rejected"
)

type Order struct {
	Id          *string   `json:"id,omitempty"`
	CryptoAsset string    `json:"crypto_asset"`
	Currency    *string   `json:"currency,omitempty"`
	Side        Side      `json:"side"`
	OrderType   OrderType `json:"order_type"`
	Price       *string   `json:"price,omitempty"`

	Amount     *string `json:"amount,omitempty"`
	FiatAmount *string `json:"fiat_amount,omitempty"`

	RefId           *string `json:"ref_id,omitempty"`
	QuoteId         *string `json:"quote_id,omitempty"`
	BankTransferRef *string `json:"bank_transfer_ref,omitempty"`

	// Response Fields
	Status *OrderStatus `json:"status,omitempty"`
}

func (c *Client) PlaceOrder(ctx context.Context, order *Order) error {
	b, err := c.post(ctx, ORDER_ENDPOINT, order)
	if err != nil {
		return fmt.Errorf("error during request: %w", err)
	}

	err = json.Unmarshal(b, order)
	if err != nil {
		return fmt.Errorf("error unmarshaling response: %w", err)
	}

	return nil
}
