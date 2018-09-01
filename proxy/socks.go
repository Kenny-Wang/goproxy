package proxy

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/shell909090/goproxy/netutil"
)

var (
	ErrProtocol        = errors.New("protocol error")
	ErrAuthMethod      = errors.New("auth method wrong")
	ErrWrongFmt        = errors.New("connect packet wrong format")
	ErrUnknownAddrType = errors.New("unknown addr type")
	ErrIPv6            = errors.New("ipv6 not support yet")
)

func readLeadByte(reader io.Reader) (b []byte, err error) {
	var c [1]byte

	n, err := reader.Read(c[:])
	if err != nil {
		return
	}
	if n < 1 {
		return nil, io.EOF
	}

	b = make([]byte, int(c[0]))
	_, err = io.ReadFull(reader, b)
	return
}

func readString(reader io.Reader) (s string, err error) {
	b, err := readLeadByte(reader)
	if err != nil {
		return
	}
	return string(b), nil
}

func GetHandshake(reader *bufio.Reader) (methods []byte, err error) {
	var c byte

	c, err = reader.ReadByte()
	if err != nil {
		return
	}
	if c != 0x05 {
		return nil, ErrProtocol
	}

	methods, err = readLeadByte(reader)
	return
}

func SendHandshakeResponse(writer *bufio.Writer, status byte) (err error) {
	_, err = writer.Write([]byte{0x05, status})
	if err != nil {
		return
	}
	return writer.Flush()
}

func GetUserPass(reader *bufio.Reader) (user string, password string, err error) {
	c, err := reader.ReadByte()
	if err != nil {
		return
	}
	if c != 0x01 {
		err = errors.New("Auth Packet Error")
		return
	}

	user, err = readString(reader)
	if err != nil {
		return
	}
	password, err = readString(reader)
	return
}

func SendAuthResult(writer *bufio.Writer, status byte) (err error) {
	var buf []byte = []byte{0x01, 0x00}

	buf[1] = status
	n, err := writer.Write(buf)
	if n != len(buf) {
		return errors.New("send buffer full")
	}
	return writer.Flush()
}

func GetConnect(reader *bufio.Reader) (hostname string, port uint16, err error) {
	var c byte

	buf := make([]byte, 3)
	_, err = io.ReadFull(reader, buf)
	if err != nil {
		return
	}
	if buf[0] != 0x05 || buf[1] != 0x01 || buf[2] != 0x00 {
		err = ErrWrongFmt
		return
	}

	c, err = reader.ReadByte()
	if err != nil {
		return
	}

	switch c {
	case 0x01: // IP V4 address
		logger.Debug("hostname in ipaddr mode.")
		var buf [4]byte
		_, err = io.ReadFull(reader, buf[:])
		if err != nil {
			return
		}
		ip := net.IP(buf[:])
		hostname = ip.String()
	case 0x03: // DOMAINNAME
		logger.Debug("hostname in domain mode.")
		hostname, err = readString(reader)
		if err != nil {
			return
		}
	case 0x04: // IP V6 address
		err = ErrIPv6
		logger.Error(err.Error())
		return
	default:
		err = ErrUnknownAddrType
		logger.Error(err.Error())
		return
	}

	err = binary.Read(reader, binary.BigEndian, &port)
	return
}

func SendConnectResponse(writer *bufio.Writer, res byte) (err error) {
	var buf []byte = []byte{0x05, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	// TODO: fix it if bind addr and port are needed.
	// REF: https://zh.wikipedia.org/wiki/SOCKS
	var n int

	buf[1] = res
	n, err = writer.Write(buf)
	if n != len(buf) {
		return io.ErrShortWrite
	}
	return writer.Flush()
}

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

func (p *SocksProxy) Start() {
	go netutil.ListenAndServe("tcp", p.listenaddr, p.ServeConn)
}

func (p *SocksProxy) ServeConn(conn net.Conn) {
	defer conn.Close()

	dstconn, err := p.SocksHandler(conn)
	if err != nil {
		return
	}

	netutil.CopyLink(conn, dstconn)
	return
}

func (p *SocksProxy) SocksHandler(conn net.Conn) (dstconn net.Conn, err error) {
	logger.Debugf("connection come from: %s => %s",
		conn.RemoteAddr(), conn.LocalAddr())

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	methods, err := GetHandshake(reader)
	if err != nil {
		return
	}

	method := byte(0xff)
	for _, m := range methods {
		if m == 0 {
			method = 0
		}
	}
	// TODO: username & password
	SendHandshakeResponse(writer, method)
	if method == 0xff {
		err = ErrAuthMethod
		logger.Error(err.Error())
		return
	}
	logger.Debug("socks handshark ok")

	hostname, port, err := GetConnect(reader)
	if err != nil {
		// general SOCKS server failure
		SendConnectResponse(writer, 0x01)
		return
	}
	logger.Debugf("dst: %s:%d", hostname, port)

	dstconn, err = p.dialer.Dial("tcp", fmt.Sprintf("%s:%d", hostname, port))
	if err != nil {
		// Connection refused
		SendConnectResponse(writer, 0x05)
		return
	}
	SendConnectResponse(writer, 0x00)

	return dstconn, nil
}
