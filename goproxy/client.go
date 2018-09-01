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

type ClientConfig struct {
	Config
	DirectRoutes     string
	ProhibitedRoutes string

	MinSess int
	MaxConn int
	Servers []*ServerDefine

	HttpUser      string
	HttpPassword  string
	Socks         string
	SocksUser     string
	SocksPassword string

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

func httpserver(addr string, handler http.Handler) {
	for {
		err := http.ListenAndServe(addr, handler)
		if err != nil {
			logger.Error("%s", err.Error())
			return
		}
	}
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

func RunClientProxy(cfg *ClientConfig) (err error) {
	var dialer netutil.Dialer
	pool := connpool.NewDialer(cfg.MinSess, cfg.MaxConn)

	for _, srv := range cfg.Servers {
		dialer, err = srv.MakeDialer()
		if err != nil {
			return
		}
		creator := tunnel.NewDialerCreator(
			dialer, "tcp4", srv.Server, srv.Username, srv.Password)
		pool.AddDialerCreator(creator)
	}

	dialer = pool

	if cfg.DnsNet == "internal" {
		dns.DefaultResolver = dns.NewTcpClient(dialer)
	}

	if cfg.DnsServer != "" {
		go RunDnsServer(cfg.DnsServer)
	}

	if cfg.AdminIface != "" {
		mux := http.NewServeMux()
		pool.Register(mux)
		go httpserver(cfg.AdminIface, mux)
	}

	if cfg.DirectRoutes != "" || cfg.ProhibitedRoutes != "" {
		fdialer := ipfilter.NewFilteredDialer(dialer)
		dialer = fdialer

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
	}

	// FIXME: port mapper?
	for _, pm := range cfg.Portmaps {
		go portmapper.CreatePortmap(pm, dialer)
	}

	if cfg.Socks != "" {
		p := proxy.NewSocksProxy(dialer, cfg.Socks, cfg.SocksUser, cfg.SocksPassword)
		p.Start()
	}

	p := proxy.NewHttpProxy(dialer, cfg.HttpUser, cfg.HttpPassword)
	logger.Infof("http start in %s", cfg.Listen)
	return http.ListenAndServe(cfg.Listen, p)
}
