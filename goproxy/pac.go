package main

import (
	"bytes"
	"html/template"
	"io/ioutil"

	"github.com/shell909090/goproxy/proxy"
)

const str_default_pac = `function FindProxyForURL(url, host) {
    return "PROXY {{.Http}}; SOCKS {{.Socks}}";
}
`

func CreatePAC(cfg *ClientConfig) (s *proxy.ServeFile, err error) {
	if cfg.PACFile != "" {
		var b []byte
		b, err = ioutil.ReadFile(cfg.PACFile)
		if err != nil {
			return
		}
		return proxy.NewServeFile(b), nil
	}
	tmpl_pac, err := template.New("main").Parse(str_default_pac)
	if err != nil {
		return
	}

	buf := bytes.NewBuffer(nil)
	err = tmpl_pac.Execute(buf, cfg)
	if err != nil {
		return
	}
	return proxy.NewServeFile(buf.Bytes()), nil
}
