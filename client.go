package hyphe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gorilla/websocket"
)

type Client struct {
	apiKey string
}

const (
	HYPHE_WEBSOCKET_CHANNEL     = "wss://sandbox-prime-access.hyphe.com/v1/websocket"
	HYPHE_AUTHENTICATION_ACTION = "authenticate"
	HYPHE_AUTHENTICATION_EVENT  = "authenticated"
	HYPHE_SUBSCRIPTION_ACTION   = "subscribe"
	HYPHE_SUBSCRIPTION_EVENT    = "subscribed"
)

func NewClient(apiKey string) *Client {
	c := &Client{
		apiKey: apiKey,
	}

	return c
}

type authenticationMessage struct {
	Action string `json:"action"`
	Key    string `json:"key"`
}

type subscribeMessage struct {
	Action   string    `json:"action"`
	Channels []Channel `json:"channels"`
}
type Channel struct {
	Name    string   `json:"name"`
	Markets []string `json:"markets"`
}

type websocketResponse struct {
	Event *string    `json:"event"`
	Bids  [][]string `json:"bids"`
	Asks  [][]string `json:"asks"`
}

func (c *Client) connect(ctx context.Context, auth *authenticationMessage, msg *subscribeMessage) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, HYPHE_WEBSOCKET_CHANNEL, nil)
	if err != nil {
		return nil, err
	}

	err = conn.WriteJSON(auth)
	if err != nil {
		fmt.Println(err)
		conn.Close()
		return nil, err
	}

	err = getAuthenticationResult(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// sum up any errors
	err = func() error {
		err := conn.WriteJSON(msg)
		if err != nil {
			return err
		}

		return getSubscriptionResult(conn)
	}()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func getAuthenticationResult(conn *websocket.Conn) error {
	for {
		fmt.Println("Receiving data from websocket...")
		_, data, err := conn.ReadMessage()
		if err == io.EOF {
			conn.Close()
			return err
		}
		if err != nil {
			fmt.Printf("error during websocket connection: %v\n", err)
			conn.Close()
			return err
		}

		msg := new(websocketResponse)
		err = json.Unmarshal(data, msg)
		if err != nil {
			return err
		}

		switch *msg.Event {
		case HYPHE_AUTHENTICATION_EVENT:
			return nil
		default:
			return fmt.Errorf("unhandled type: %v", *msg.Event)
		}
	}
}

func getSubscriptionResult(conn *websocket.Conn) error {
	for {
		fmt.Println("Receiving data from websocket...")
		_, data, err := conn.ReadMessage()
		if err == io.EOF {
			return err
		}
		if err != nil {
			fmt.Printf("error during websocket connection: %v\n", err)
			return err
		}

		msg := new(websocketResponse)
		err = json.Unmarshal(data, msg)
		if err != nil {
			return err
		}

		switch *msg.Event {
		case HYPHE_AUTHENTICATION_EVENT:
			continue
		case HYPHE_SUBSCRIPTION_EVENT:
			return nil
		default:
			return fmt.Errorf("unhandled event: %v", *msg.Event)
		}
	}
}
