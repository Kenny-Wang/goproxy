package proxy

import (
	"encoding/base64"
	"net/http"
	"strings"
)

type HttpBasicAuth struct {
	handler    http.Handler
	ReqHeader  string
	RespHeader string
	RespStatus int
	auths      map[string]string
}

func NewHttpBasicAuth(upstream http.Handler) (bauth *HttpBasicAuth) {
	bauth = &HttpBasicAuth{
		handler:    upstream,
		ReqHeader:  "Authorization",
		RespHeader: "WWW-Authenticate",
		RespStatus: http.StatusUnauthorized,
		auths:      make(map[string]string),
	}
	return
}

func NewProxyBasicAuth(upstream http.Handler) (bauth *HttpBasicAuth) {
	bauth = &HttpBasicAuth{
		handler:    upstream,
		ReqHeader:  "Proxy-Authorization",
		RespHeader: "Proxy-Authenticate",
		RespStatus: http.StatusProxyAuthRequired,
		auths:      make(map[string]string),
	}
	return
}

func (bauth *HttpBasicAuth) AddUserPass(usr, pwd string) {
	bauth.auths[usr] = pwd
}

func (bauth *HttpBasicAuth) Authenticate(req *http.Request) bool {
	auth := req.Header[bauth.ReqHeader]
	if auth == nil || len(auth) == 0 {
		return false
	}

	realm := strings.SplitN(auth[0], " ", 2)
	if len(realm) != 2 || realm[0] != "Basic" {
		return false
	}

	payload, err := base64.StdEncoding.DecodeString(realm[1])
	if err != nil {
		return false
	}

	usrpwd := strings.SplitN(string(payload), ":", 2)
	if len(usrpwd) != 2 {
		return false
	}

	if pwd, ok := bauth.auths[usrpwd[0]]; ok {
		return usrpwd[1] == pwd
	}
	return false
}

func (bauth *HttpBasicAuth) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !bauth.Authenticate(req) {
		w.Header().Set(bauth.RespHeader, "Basic realm=\"GoProxy\"")
		http.Error(w, http.StatusText(bauth.RespStatus), bauth.RespStatus)
		return
	}

	bauth.handler.ServeHTTP(w, req)
	return
}
