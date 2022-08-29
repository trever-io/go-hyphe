package hyphe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gorilla/websocket"
)

const PRICES_ENDPOINT = "/prices"

type CurrencyPrices struct {
	Currency     string                  `json:"currency"`
	CryptoAssets map[string]*CryptoAsset `json:"crypto_assets"`
}

type CryptoAsset struct {
	Ask    json.Number `json:"ask"`
	AskVol json.Number `json:"ask_vol"`
	Bid    json.Number `json:"bid"`
	BidVol json.Number `json:"bid_vol"`
}

type OrderBook struct {
	Asks [][]string
	Bids [][]string
}

const HYPHE_PRICE_CHANNEL_NAME = "prices"
const HYPHE_PRICE_EVENT = "price"

func (c *Client) Prices(ctx context.Context) (*CurrencyPrices, error) {
	b, err := c.get(ctx, PRICES_ENDPOINT)
	if err != nil {
		return nil, fmt.Errorf("error during request: %w", err)
	}

	result := new(CurrencyPrices)
	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	return result, nil
}

func (c *Client) GetPriceEvent(ctx context.Context, baseAsset string, quoteAsset string) (*OrderBook, error) {
	symbol := fmt.Sprintf("%v-%v", baseAsset, quoteAsset)

	auth := authenticationMessage{
		Action: HYPHE_AUTHENTICATION_ACTION,
		Key:    c.apiKey,
	}

	sub := subscribeMessage{
		Action:   HYPHE_SUBSCRIPTION_ACTION,
		Channels: []Channel{{Name: HYPHE_PRICE_CHANNEL_NAME, Markets: []string{symbol}}},
	}

	conn, err := c.connect(ctx, &auth, &sub)
	if err != nil {
		return nil, fmt.Errorf("error connecting to websocket: %w", err)
	}
	defer conn.Close()

	book, err := getOrderBookSnapshot(conn)
	if err != nil {
		return nil, err
	}

	return book, nil

}

func getOrderBookSnapshot(conn *websocket.Conn) (*OrderBook, error) {
	for {
		_, data, err := conn.ReadMessage()
		if err == io.EOF {
			return nil, err
		}
		if err != nil {
			tmp := fmt.Errorf("error during websocket connection: %w", err)
			fmt.Println(tmp)
			return nil, tmp
		}

		msg := new(websocketResponse)
		err = json.Unmarshal(data, msg)
		if err != nil {
			return nil, err
		}

		var book *OrderBook
		if *msg.Event == HYPHE_PRICE_EVENT {
			asks := msg.Asks
			bids := msg.Bids

			book = &OrderBook{Asks: asks, Bids: bids}
		} else {
			return nil, fmt.Errorf("unexpected event: %v", *msg.Event)
		}

		return book, nil
	}
}
