package mongodb

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	client *mongo.Client
	db     *mongo.Database
}

func New(ctx context.Context, user string, password string, host string, port int, database string) (*Client, error) {
	opts := options.Client().ApplyURI("mongodb://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/")
	if user != "" || password != "" {
		opts.SetAuth(options.Credential{
			Username:   user,
			Password:   password,
			AuthSource: "admin",
		})
	}
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	return &Client{
		client: client,
		db:     client.Database(database),
	}, nil
}

func (c *Client) DB() *mongo.Database {
	return c.db
}

func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}
