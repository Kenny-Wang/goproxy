package proxy

import (
	"context"
	"net"

	socks5 "github.com/armon/go-socks5"
	"github.com/shell909090/goproxy/netutil"
)

type SocksProxy struct {
	dialer     netutil.Dialer
	listenaddr string
	username   string
	password   string
}

func NewSocksProxy(dialer netutil.Dialer, addr, username, password string) (p *SocksProxy) {
	p = &SocksProxy{
		dialer:     dialer,
		listenaddr: addr,
		username:   username,
		password:   password,
	}
	if username != "" && password != "" {
		logger.Info("socks5 proxy auth required")
	}
	return
}

func (p *SocksProxy) Start() (err error) {
	conf := &socks5.Config{
		Dial: p.Dial,
	}
	server, err := socks5.New(conf)
	if err != nil {
		logger.Error("fail to start socks5 server:", err.Error())
		return
	}

	go func() {
		logger.Info("socks server started")
		err := server.ListenAndServe("tcp4", p.listenaddr)
		if err != nil {
			logger.Error(err.Error())
		}
		// If there have an error, socks will shutdown and not reboot.
		// Let's see if there have any problem.
		// Auto-reboot may cause loop, which I'm trying to avoid.
	}()
	return
}

func (p *SocksProxy) Dial(ctx context.Context, network, addr string) (conn net.Conn, err error) {
	logger.Info("connect to", network, addr)
	conn, err = p.dialer.Dial(network, addr)
	return
}
