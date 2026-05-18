package neo4j

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Client struct {
	driver neo4j.DriverWithContext
}

func NewClient(ctx context.Context, url string, user string, password string) (*Client, error) {
	var auth neo4j.AuthToken
	if user == "" && password == "" {
		auth = neo4j.NoAuth()
	} else {
		auth = neo4j.BasicAuth(user, password, "")
	}

	driver, err := neo4j.NewDriverWithContext(url, auth)
	if err != nil {
		return nil, fmt.Errorf("create neo4j driver: %w", err)
	}

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("verify neo4j connectivity: %w", err)
	}

	return &Client{driver: driver}, nil
}

func (c *Client) Driver() neo4j.DriverWithContext {
	return c.driver
}

func (c *Client) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}
