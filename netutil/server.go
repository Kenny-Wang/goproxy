package netutil

import "net"

func ListenAndServe(network, address string, handler func(net.Conn)) (err error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer listener.Close()
	logger.Debugf("listen in %s %s", network, address)

	var conn net.Conn
	for {
		conn, err = listener.Accept()
		if err != nil {
			logger.Error(err.Error())
			return
		}
		go handler(conn)
	}
}
