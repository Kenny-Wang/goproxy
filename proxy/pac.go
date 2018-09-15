package proxy

import "net/http"

type ServeFile struct {
	body string
}

func NewServeFile(body string) (s *ServeFile) {
	s = &ServeFile{
		body: body,
	}
	return
}

func (s *ServeFile) ServeHTTP(w http.ResponseWriter, req *http.Request) {
}
