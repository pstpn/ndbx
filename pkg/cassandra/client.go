package cassandra

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
)

type Client struct {
	session *gocql.Session
}

const (
	defaultTimeoutSeconds        = 10
	defaultConnectTimeoutSeconds = 10
)

func NewClient(
	ctx context.Context,
	hosts []string,
	port int,
	username string,
	password string,
	keyspace string,
	consistency string,
) (*Client, error) {
	cleanKeyspace := strings.Trim(strings.TrimSpace(keyspace), "\"'")
	if cleanKeyspace == "" {
		return nil, errors.New("empty cassandra keyspace")
	}

	sysCluster := newCluster(hosts, port, username, password, consistency)
	sysSession, err := sysCluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("create cassandra system session: %w", err)
	}

	createKeyspaceQuery := fmt.Sprintf(
		"CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class':'SimpleStrategy','replication_factor':1}",
		cleanKeyspace,
	)
	if err := sysSession.Query(createKeyspaceQuery).WithContext(ctx).Exec(); err != nil {
		sysSession.Close()
		return nil, fmt.Errorf("create cassandra keyspace: %w", err)
	}
	sysSession.Close()

	cluster := newCluster(hosts, port, username, password, consistency)
	cluster.Keyspace = cleanKeyspace
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("create cassandra session: %w", err)
	}

	if err := session.Query("SELECT now() FROM system.local").WithContext(ctx).Exec(); err != nil {
		session.Close()
		return nil, fmt.Errorf("ping cassandra: %w", err)
	}

	return &Client{session: session}, nil
}

func (c *Client) Session() *gocql.Session {
	return c.session
}

func (c *Client) Close() {
	if c != nil && c.session != nil {
		c.session.Close()
	}
}

func newCluster(hosts []string, port int, username string, password string, consistency string) *gocql.ClusterConfig {
	cluster := gocql.NewCluster(hosts...)
	cluster.Port = port
	cluster.Timeout = defaultTimeoutSeconds * time.Second
	cluster.ConnectTimeout = defaultConnectTimeoutSeconds * time.Second
	cluster.DisableInitialHostLookup = true
	cluster.Consistency = parseConsistency(consistency)
	if username != "" || password != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{Username: username, Password: password}
	}

	return cluster
}

func parseConsistency(level string) gocql.Consistency {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "ALL":
		return gocql.All
	case "ANY":
		return gocql.Any
	case "EACH_QUORUM":
		return gocql.EachQuorum
	case "LOCAL_ONE":
		return gocql.LocalOne
	case "LOCAL_QUORUM":
		return gocql.LocalQuorum
	case "ONE":
		return gocql.One
	case "QUORUM":
		return gocql.Quorum
	case "THREE":
		return gocql.Three
	case "TWO":
		return gocql.Two
	default:
		return gocql.One
	}
}
