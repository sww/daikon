package kumo

import (
	"fmt"
	"log"
	"sync"

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

	wait := new(sync.WaitGroup)
	wait.Add(connections)

	var errors []error

	for i := 0; i < connections; i++ {
		go func() {
			defer wait.Done()

			connection := new(Connection)
			client, err := nntpclient.New("tcp", fmt.Sprintf("%v:%v", server, port))
			if err != nil {
				log.Printf("Error Connecting to \"%v\"", server)
				errors = append(errors, err)
			}

			msg, err := client.Authenticate(username, password)
			if err != nil {
				log.Printf("Problem authenticating, got msg: %v", msg)
				errors = append(errors, err)
			}

			connection.group = ""
			connection.client = client

			pool.connections <- *connection
		}()
	}

	wait.Wait()

	if len(errors) > 0 {
		// Return an error if any connection failed for now.
		return nil, errors[0]
	} else {
		return pool, nil
	}
}
