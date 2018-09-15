package main

import (
	"net/http"
	"strings"

	"github.com/shell909090/goproxy/connpool"
	"github.com/shell909090/goproxy/cryptconn"
	"github.com/shell909090/goproxy/dns"
	"github.com/shell909090/goproxy/ipfilter"
	"github.com/shell909090/goproxy/netutil"
	"github.com/shell909090/goproxy/portmapper"
	"github.com/shell909090/goproxy/proxy"
	"github.com/shell909090/goproxy/tunnel"
)

type ServerDefine struct {
	Server      string
	CryptMode   string
	RootCAs     string
	CertFile    string
	CertKeyFile string
	Cipher      string
	Key         string
	Username    string
	Password    string
}

func (sd *ServerDefine) MakeDialer() (dialer netutil.Dialer, err error) {
	if strings.ToLower(sd.CryptMode) == "tls" {
		dialer, err = NewTlsDialer(sd.CertFile, sd.CertKeyFile, sd.RootCAs)
	} else {
		cipher := sd.Cipher
		if cipher == "" {
			cipher = "aes"
		}
		dialer, err = cryptconn.NewDialer(netutil.DefaultTcpDialer, cipher, sd.Key)
	}
	return
}

type ClientConfig struct {
	Config
	DirectRoutes     string
	ProhibitedRoutes string

	MinSess int
	MaxConn int
	Servers []*ServerDefine

	Http        string
	HttpUser    string
	HttpPwd     string
	PACFile     string
	Admin       string
	HttpAdmin   int
	AdminUser   string
	AdminPwd    string
	Socks       string
	SocksUser   string
	SocksPwd    string
	Transparent string

	Portmaps  []portmapper.PortMap
	DnsServer string
}

func LoadClientConfig(basecfg *Config) (cfg *ClientConfig, err error) {
	err = LoadJson(ConfigFile, &cfg)
	if err != nil {
		return
	}
	cfg.Config = *basecfg
	if cfg.MaxConn == 0 {
		cfg.MaxConn = 16
	}
	return
}

func MakeDialer(cfg *ClientConfig) (pooldialer *connpool.Dialer, err error) {
	var dialer netutil.Dialer
	pooldialer = connpool.NewDialer(cfg.MinSess, cfg.MaxConn)
	for _, srv := range cfg.Servers {
		dialer, err = srv.MakeDialer()
		if err != nil {
			return
		}
		creator := tunnel.NewDialerCreator(
			dialer, "tcp4", srv.Server, srv.Username, srv.Password)
		pooldialer.AddDialerCreator(creator)
	}
	return
}

func MakeFilteredDialer(dialer netutil.Dialer, cfg *ClientConfig) (fdialer *ipfilter.FilteredDialer, err error) {
	fdialer = ipfilter.NewFilteredDialer(dialer)

	// push first, work first. prohibited should been setup at first.
	if cfg.ProhibitedRoutes != "" {
		err = fdialer.LoadFilter(netutil.DefaultFalseDialer, cfg.ProhibitedRoutes)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}

	if cfg.DirectRoutes != "" {
		err = fdialer.LoadFilter(netutil.DefaultTcpDialer, cfg.DirectRoutes)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}
	return
}

func RunClientProxy(cfg *ClientConfig) (err error) {
	var dialer netutil.Dialer

	if cfg.Http == "" && cfg.Socks == "" && cfg.Transparent == "" {
		logger.Critical("You don't wanna run any client mode. I quit.")
		return
	}

	pooldialer, err := MakeDialer(cfg)
	if err != nil {
		return
	}
	dialer = pooldialer

	if cfg.DnsNet == "internal" {
		dns.DefaultResolver = dns.NewTcpClient(dialer)
	}

	if cfg.DirectRoutes != "" || cfg.ProhibitedRoutes != "" {
		dialer, err = MakeFilteredDialer(dialer, cfg)
		if err != nil {
			return
		}
	}

	// FIXME: port mapper?
	for _, pm := range cfg.Portmaps {
		go portmapper.CreatePortmap(pm, dialer)
	}

	if cfg.DnsServer != "" {
		go RunDnsServer(cfg.DnsServer)
	}

	if cfg.Socks != "" {
		p := proxy.NewSocksProxy(dialer, cfg.SocksUser, cfg.SocksPwd)
		p.Start(cfg.Socks)
	}

	if cfg.Transparent != "" {
		p := proxy.NewTransparentProxy(dialer)
		p.Start(cfg.Transparent)
	}

	if cfg.Admin != "" {
		handler := MakeAdminHandler(
			pooldialer.Pool, cfg.AdminUser, cfg.AdminPwd)
		go HttpListenAndServer(cfg.Admin, handler)
	}

	if cfg.Http != "" {
		httpproxy := proxy.NewHttpProxy(dialer, cfg.HttpUser, cfg.HttpPwd)

		if cfg.HttpAdmin != 0 {
			httpproxy.Handler = MakeAdminHandler(
				pooldialer.Pool, cfg.AdminUser, cfg.AdminPwd)
		}

		mux := http.NewServeMux()

		var pac http.Handler
		pac, err = CreatePAC(cfg)
		if err != nil {
			logger.Error(err.Error())
			return
		}
		mux.Handle("/pac.json", pac)

		if httpproxy.Handler != nil {
			mux.Handle("/", httpproxy.Handler)
		}
		httpproxy.Handler = mux

		httpproxy.Start(cfg.Http)
	}
	select {}
	return
}
