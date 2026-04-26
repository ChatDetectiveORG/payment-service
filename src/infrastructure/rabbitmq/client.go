package rabbitmq

import (
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	url string
	cfg amqp.Config

	mu   sync.Mutex
	conn *amqp.Connection
}

func NewClient(url string, cfg amqp.Config) (*Client, error) {
	client := &Client{url: url, cfg: cfg}
	if err := client.connect(); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && !c.conn.IsClosed() {
		return nil
	}

	conn, err := amqp.DialConfig(c.url, c.cfg)
	if err != nil {
		return fmt.Errorf("amqp dial: %w", err)
	}
	c.conn = conn
	return nil
}

func (c *Client) Conn() (*amqp.Connection, error) {
	if err := c.connect(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn, nil
}

func (c *Client) Channel() (*amqp.Channel, error) {
	conn, err := c.Conn()
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open channel: %w", err)
	}
	return ch, nil
}
