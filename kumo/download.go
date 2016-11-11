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
	Queue          chan Segment
	ConnectionPool ConnectionPool
	DecodeQueue    chan string
	Logger         *dumblog.DumbLog
	Progress       *Progress
	Wait           *sync.WaitGroup
	TempPath       string
}

func InitDownload(host, username, password string, port, connections int, ssl bool, w *sync.WaitGroup) (*Download, error) {
	connectionPool, err := InitConnectionPool(host, username, password, port, connections, ssl)
	if err != nil {
		return nil, err
	}

	return &Download{
		ConnectionPool: *connectionPool,
		Queue:          make(chan Segment),
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
			d.Logger.Print("[DOWNLOAD] Run() got segment ", segment)
			go func(segment Segment) {
				defer d.Progress.Add(segment.Bytes)

				connection := <-d.ConnectionPool.connections
				segmentName, err := d.download(segment.Segment, segment.Group, &connection)

				if err != nil {
					d.Progress.addBroken(segment.Bytes)
					d.Logger.Printf("[DOWNLOAD] Done() because of err: %v", err)
					d.Wait.Done()
					return
				}

				d.DecodeQueue <- segmentName
			}(segment)
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

func (d *Download) download(segmentName, segmentGroup string, connection *Connection) (string, error) {
	defer func() { d.ConnectionPool.connections <- *connection }()

	d.Logger.Printf("[DOWNLOAD] download(%v)", segmentName)
	if segmentGroup != connection.group {
		d.Logger.Printf("[DOWNLOAD] Switching from group '%v' to '%v'", connection.group, segmentGroup)
		// TODO: Handle error.
		group(segmentGroup, connection)
	}

	_, _, resp, err := connection.client.Body(fmt.Sprintf("<%s>", segmentName))
	if err != nil {
		return "", err
	}

	msg, err := ioutil.ReadAll(resp)
	if err != nil {
		return "", err
	}

	fullSegment := filepath.Join(d.TempPath, segmentName)
	ioutil.WriteFile(fullSegment, msg, 0644)

	d.Logger.Printf("[DOWNLOAD] download() wrote '%v'", fullSegment)

	return fullSegment, nil
}
