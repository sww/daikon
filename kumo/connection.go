package kumo

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/sww/kumo/nntp"
)

type Connection struct {
	group  string
	client *nntp.NNTP
}

type ConnectionPool struct {
	size        int
	connections chan Connection
}

func InitConnectionPool(server, username, password string, port, connections int, ssl bool) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		connections: make(chan Connection, connections),
		size:        connections,
	}

	wait := new(sync.WaitGroup)
	wait.Add(connections)

	for i := 0; i < connections; i++ {
		go func() {
			defer wait.Done()

			connection := new(Connection)
			client, err := nntp.New("tcp", fmt.Sprintf("%v:%v", server, port), ssl)
			if err != nil {
				log.Printf("Error Connecting to \"%v\"", server)
				return
			}

			msg, err := client.Auth(username, password)
			if err != nil {
				log.Printf("Problem authenticating, got msg: %v", msg)
				return
			}

			connection.group = ""
			connection.client = client

			pool.connections <- *connection
		}()
	}

	wait.Wait()

	if len(pool.connections) < 1 {
		return nil, errors.New("no connections available")
	}

	return pool, nil
}
