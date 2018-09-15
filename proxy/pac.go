package proxy

import "net/http"

type ServeFile struct {
	body []byte
}

func NewServeFile(body []byte) (s *ServeFile) {
	s = &ServeFile{
		body: body,
	}
	return
}

func (s *ServeFile) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(s.body)
	if err != nil {
		logger.Error(err.Error())
		return
	}
}

var DEFAULT_PAC = `
abcdef
`

func NewDefaultPAC() (s *ServeFile) {
	return NewServeFile([]byte(DEFAULT_PAC))
}
