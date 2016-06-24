package daikon

import (
	"fmt"
	"log"

	"github.com/dustin/go-nntp/client"
)

type Connection struct {
	group  string
	client *nntpclient.Client
}

type ConnectionPool struct {
	size        int
	connections chan Connection
}

func InitConnectionPool(server, username, password string, port, connections int) (*ConnectionPool, error) {
	pool := new(ConnectionPool)
	pool.connections = make(chan Connection, connections)
	pool.size = connections

	for i := 0; i < connections; i++ {
		connection := new(Connection)
		client, err := nntpclient.New("tcp", fmt.Sprintf("%v:%v", server, port))
		if err != nil {
			log.Printf("Error Connecting to \"%v\"", server)
			return nil, err
		}

		msg, err := client.Authenticate(username, password)
		if err != nil {
			log.Printf("Problem authenticating, got msg: %v", msg)
			return nil, err
		}

		connection.group = ""
		connection.client = client

		pool.connections <- *connection
	}

	return pool, nil
}
