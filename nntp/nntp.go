package nntp

import (
	"io"
	"net/textproto"
)

func New(net, addr string) (*NNTP, error) {
	conn, err := textproto.Dial(net, addr)
	if err != nil {
		return nil, err
	}

	_, _, err = conn.ReadCodeLine(200)
	if err != nil {
		return nil, err
	}

	nntp := NNTP{
		conn: conn,
	}

	return &nntp, nil
}

type NNTP struct {
	conn *textproto.Conn
}

func (n *NNTP) Auth(user, password string) (string, error) {
	if err := n.conn.PrintfLine("authinfo user %s", user); err != nil {
		return "", err
	}

	_, msg, err := n.conn.ReadCodeLine(381)
	if err != nil {
		return "", err
	}

	err = n.conn.PrintfLine("authinfo pass %s", password)
	if err != nil {
		return "", err
	}

	_, msg, err = n.conn.ReadCodeLine(281)
	if err != nil {
		return "", err
	}

	return msg, nil
}

func (n *NNTP) Group(group string) (string, error) {
	err := n.conn.PrintfLine("GROUP %s", group)
	if err != nil {
		return "", err
	}

	_, msg, err := n.conn.ReadCodeLine(211)
	if err != nil {
		return "", err
	}

	return msg, nil
}

func (n *NNTP) Article(id string) (string, error) {
	err := n.conn.PrintfLine("ARTICLE %s", id)
	if err != nil {
		return "", err
	}

	_, msg, err := n.conn.ReadCodeLine(220)
	if err != nil {
		return "", err
	}

	return msg, nil
}

func (n *NNTP) Body(id string) (int, string, io.Reader, error) {
	err := n.conn.PrintfLine("BODY %s", id)
	if err != nil {
		return 0, "", nil, err
	}

	code, msg, err := n.conn.ReadCodeLine(22)
	if err != nil {
		return 0, "", nil, err
	}

	reader := n.conn.DotReader()

	return code, msg, reader, nil
}
