package main

import (
	"net/http"

	logging "github.com/op/go-logging"
	"github.com/shell909090/goproxy/connpool"
	"github.com/shell909090/goproxy/proxy"
)

var logger = logging.MustGetLogger("")

func HttpListenAndServer(addr string, handler http.Handler) {
	for {
		err := http.ListenAndServe(addr, handler)
		if err != nil {
			logger.Error("%s", err.Error())
			return
		}
	}
}

func MakeAdminHandler(pool *connpool.Pool, user, pwd string) (handler http.Handler) {
	mux := http.NewServeMux()
	pool.Register(mux)
	if user != "" && pwd != "" {
		bauth := proxy.NewHttpBasicAuth(mux)
		bauth.AddUserPass(user, pwd)
		return bauth
	}
	return mux
}
