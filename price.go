package hyphe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gorilla/websocket"
)

type LiquidityBook struct {
	Asks [][]string
	Bids [][]string
}

const HYPHE_PRICE_CHANNEL_NAME = "prices"
const HYPHE_PRICE_EVENT = "price"

func (c *Client) GetPriceEvent(ctx context.Context, baseAsset string, quoteAsset string) (*LiquidityBook, error) {
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

	book, err := getLiquidityBookSnapshot(conn)
	if err != nil {
		return nil, err
	}

	return book, nil

}

func getLiquidityBookSnapshot(conn *websocket.Conn) (*LiquidityBook, error) {
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

		var book *LiquidityBook
		if *msg.Event == HYPHE_PRICE_EVENT {
			asks := msg.Asks
			bids := msg.Bids

			book = &LiquidityBook{Asks: asks, Bids: bids}
		} else {
			return nil, fmt.Errorf("unexpected event: %v", *msg.Event)
		}

		return book, nil
	}
}
