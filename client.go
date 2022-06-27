package hyphe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
	apiUrl string
	apiKey string
}

var (
	ErrNoApiKey = errors.New("no api key specified")
)

const (
	HYPHE_WEBSOCKET_CHANNEL     = "wss://sandbox-prime-access.hyphe.com/v1/websocket"
	HYPHE_AUTHENTICATION_ACTION = "authenticate"
	HYPHE_AUTHENTICATION_EVENT  = "authenticated"
	HYPHE_SUBSCRIPTION_ACTION   = "subscribe"
	HYPHE_SUBSCRIPTION_EVENT    = "subscribed"
)

const (
	REST_URL         = "https://prime-access.hyphe.com/v1"
	REST_SANDBOX_URL = "https://sandbox-prime-access.hyphe.com/v1"

	Authorization_HEADER = "Authorization"
	TOO_MANY_REQUESTS    = "too many requests"
)

var Debug = false

type ApiError struct {
	Message string         `json:"error"`
	Errors  map[string]any `json:"errors"`
	Code    int            `json:"-"`
}

func newApiError(code int) *ApiError {
	return &ApiError{
		Code: code,
	}
}

func (e *ApiError) Error() string {
	return fmt.Sprintf("API Error: Code(%d) %v", e.Code, e.Message)
}

func NewClient(apiKey string) *Client {
	c := &Client{
		apiUrl: REST_URL,
		apiKey: apiKey,
	}

	return c
}

func (c *Client) Sandbox() {
	c.apiUrl = REST_SANDBOX_URL
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

func (c *Client) post(ctx context.Context, endpoint string, request any) ([]byte, error) {
	if c.apiKey == "" {
		return nil, ErrNoApiKey
	}

	b, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	body := bytes.NewReader(b)
	url := fmt.Sprintf("%v%v", c.apiUrl, endpoint)

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add(Authorization_HEADER, fmt.Sprintf("Bearer %v", c.apiKey))
	req.Header.Add("content-type", "application/json")

	return c.doRequest(req)
}

func (c *Client) get(ctx context.Context, endpoint string) ([]byte, error) {
	if c.apiKey == "" {
		return nil, ErrNoApiKey
	}

	url := fmt.Sprintf("%v%v", c.apiUrl, endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Add(Authorization_HEADER, fmt.Sprintf("Bearer %v", c.apiKey))
	req.Header.Add("content-type", "application/json")

	return c.doRequest(req)
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting response: %w", err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if Debug {
		log.Println(string(b))
	}

	if resp.StatusCode >= 300 {
		apiErr := newApiError(resp.StatusCode)

		if resp.StatusCode == http.StatusTooManyRequests {
			apiErr.Message = TOO_MANY_REQUESTS
		}

		err := json.Unmarshal(b, apiErr)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling response body: %w", err)
		}

		return nil, apiErr
	}

	return b, nil
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
