package tunnel

import (
	"fmt"
	"net"
	"time"

	"github.com/shell909090/goproxy/netutil"
)

type ClientCreator struct {
	netutil.ConnCreator
	username string
	password string
}

func NewClientCreator(creator netutil.ConnCreator, username, password string) (cc *ClientCreator) {
	return &ClientCreator{
		ConnCreator: creator,
		username:    username,
		password:    password,
	}
}

func (cc *ClientCreator) Create() (client *Client, err error) {
	conn, err := cc.ConnCreator.CreateConn()
	if err != nil {
		logger.Error(err.Error())
		return
	}

	ti := time.AfterFunc(AUTH_TIMEOUT*time.Millisecond, func() {
		logger.Errorf("auth timeout %s.", conn.RemoteAddr())
		conn.Close()
	})
	defer ti.Stop()

	if cc.username != "" || cc.password != "" {
		logger.Noticef("auth with username: %s, password: %s.",
			cc.username, cc.password)
	}

	auth := Auth{
		Username: cc.username,
		Password: cc.password,
	}
	err = WriteFrame(conn, MSG_AUTH, 0, &auth)
	if err != nil {
		return
	}

	var errno Result
	frslt, err := ReadFrame(conn, &errno)
	if err != nil {
		return
	}

	if frslt.Header.Type != MSG_RESULT {
		return nil, ErrUnexpectedPkg
	}
	if errno != ERR_NONE {
		conn.Close()
		return nil, fmt.Errorf("create connection failed with code: %d.", errno)
	}

	logger.Notice("auth passed.")
	client = NewClient(conn)
	return
}

type Client struct {
	*Fabric
}

func NewClient(conn net.Conn) (client *Client) {
	client = &Client{
		Fabric: NewFabric(conn, 0),
	}
	client.dft_fiber = client
	return
}

func (client *Client) Dial(network, address string) (conn net.Conn, err error) {
	c := NewConn(client.Fabric)
	c.streamid, err = client.Fabric.PutIntoNextId(c)
	if err != nil {
		return
	}

	logger.Debugf("%s try to dial %s:%s.", client.String(), network, address)

	err = c.Connect(network, address)
	if err != nil {
		logger.Error(err.Error())
	}
	logger.Infof("%s connected.", c.String())
	conn = c
	return
}

func (client *Client) SendFrame(f *Frame) (err error) {
	logger.Errorf("client should never recv unmapped frame: %s.", f.Debug())
	return
}

func (client *Client) CloseFiber(streamid uint16) (err error) {
	panic("client's CloseFiber should never been called.")
	return
}
