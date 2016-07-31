package kumo

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/sww/dumblog"
)

type Download struct {
	Stop           chan bool
	Queue          chan *Segment
	ConnectionPool ConnectionPool
	DecodeQueue    chan string
	Logger         *dumblog.DumbLog
	Progress       *Progress
	Wait           *sync.WaitGroup
	TempPath       string
}

func InitDownload(host, username, password string, port, connections int, w *sync.WaitGroup) (*Download, error) {
	connectionPool, err := InitConnectionPool(host, username, password, port, connections)
	if err != nil {
		return nil, err
	}

	return &Download{
		ConnectionPool: *connectionPool,
		Queue:          make(chan *Segment),
		Stop:           make(chan bool, 1),
		Wait:           w,
	}, nil
}

func (d *Download) Run() {
	for {
		select {
		case <-d.Stop:
			d.Logger.Print("Download Stopping")
			return
		case segment := <-d.Queue:
			go func() {
				defer d.Progress.Add(segment.Bytes)
				d.Logger.Print("[DOWNLOAD] Download.Run() got segment ", segment.Segment)

				connection := <-d.ConnectionPool.connections
				segmentName, err := d.download(segment, &connection)

				if err != nil {
					d.Progress.isBroken = true
					d.Wait.Done()
					d.Logger.Print("[DOWNLOAD] d.Wait.Done() because of err ", err)
					return
				}

				d.DecodeQueue <- segmentName
			}()
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func group(group string, connection *Connection) error {
	_, err := connection.client.Group(group)
	if err != nil {
		return err
	}

	connection.group = group

	return nil
}

func (d *Download) download(segment *Segment, connection *Connection) (string, error) {
	defer func() { d.ConnectionPool.connections <- *connection }()

	d.Logger.Print("[DOWNLOAD] Group: ", segment.Group)
	if segment.Group != connection.group {
		d.Logger.Printf("[DOWNLOAD] Switching from group %v to %v", connection.group, segment.Group)
		// TODO: Handle error.
		group(segment.Group, connection)
	}

	_, _, resp, err := connection.client.Body(fmt.Sprintf("<%s>", segment.Segment))
	if err != nil {
		return "", err
	}

	msg, err := ioutil.ReadAll(resp)
	if err != nil {
		return "", err
	}

	fullSegment := filepath.Join(d.TempPath, segment.Segment)
	ioutil.WriteFile(fullSegment, msg, 0644)

	d.Logger.Print("[DOWNLOAD] d.Download() wrote ", fullSegment)

	return fullSegment, nil
}
