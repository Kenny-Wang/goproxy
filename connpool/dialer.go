package connpool

import (
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/shell909090/goproxy/netutil"
	"github.com/shell909090/goproxy/tunnel"
)

type Dialer struct {
	*Pool
	MinSess  int
	MaxConn  int
	lock     sync.Mutex
	creators []*tunnel.ClientCreator
}

func NewDialer(MinSess, MaxConn int) (dialer *Dialer) {
	if MinSess == 0 {
		MinSess = 0
	}
	if MaxConn == 0 {
		MaxConn = 64
	}
	dialer = &Dialer{
		Pool:    NewPool(),
		MinSess: MinSess,
		MaxConn: MaxConn,
	}
	go dialer.loop()
	return
}

func (dialer *Dialer) AddClientCreator(orig *tunnel.ClientCreator) {
	dialer.lock.Lock()
	defer dialer.lock.Unlock()
	dialer.creators = append(dialer.creators, orig)
}

// CAUTION: balance should run after loop begin
// because creators are added one by one, it will take a while.
func (dialer *Dialer) loop() {
	for {
		time.Sleep(60 * time.Second)
		err := dialer.balance()
		if err != nil {
			logger.Error(err.Error())
		}
	}
}

func (dialer *Dialer) balance() (err error) {
	tsize := dialer.GetSize()
	if tsize < dialer.MinSess {
		logger.Info("create tunnel because tsize < minsess.")
		err = dialer.newTunnel(false)
		if err != nil {
			return
		}
	}

	_, fsize := dialer.getMinimum()
	if fsize > dialer.MaxConn {
		logger.Info("create tunnel because fsize > maxconn.")
		err = dialer.newTunnel(false)
		if err != nil {
			return
		}
	}
	return
}

// Get one or create one.
func (dialer *Dialer) Get() (tun tunnel.Tunnel, err error) {
	if dialer.GetSize() == 0 {
		err = dialer.newTunnel(true)
		if err != nil {
			return
		}
	}

	tun, _ = dialer.getMinimum()
	if tun == nil {
		err = ErrNoSession
		return
	}
	return
}

// Randomly select a server, try to connect with it. If it is failed, try next.
// Repeat for DIAL_RETRY times.
// Each time it will take 2 ^ (net.ipv4.tcp_syn_retries + 1) - 1 second(s).
// eg. net.ipv4.tcp_syn_retries = 4, connect will timeout in 2 ^ (4 + 1) -1 = 31s.
func (dialer *Dialer) newTunnel(create bool) (err error) {
	var tun tunnel.Tunnel
	dialer.lock.Lock()
	if create && (dialer.GetSize() != 0) {
		dialer.lock.Unlock()
		logger.Debug("create first tunnel but already have one.")
		return
	}

	if len(dialer.creators) == 0 {
		dialer.lock.Unlock()
		err = ErrNoCreator
		logger.Error(err.Error())
		return
	}

	start := rand.Int()
	end := start + DIAL_RETRY*len(dialer.creators)
	for i := start; i < end; i++ {
		orig := dialer.creators[i%len(dialer.creators)]
		tun, err = orig.Create()
		if err != nil {
			logger.Error(err.Error())
			continue
		}
		break
	}
	dialer.lock.Unlock()

	if err != nil {
		logger.Critical("can't connect to any server, quit.")
		return
	}
	logger.Notice("session created.")

	dialer.Add(tun)
	go dialer.sessRun(tun)
	return
}

// Don't need to check less session here.
// Mostly, less sess counter in here will not more then the counter in Get.
// The only exception is that the closing session is the one and only one
// lower then max_conn
// but we can think that as over max_conn line just happened.
func (dialer *Dialer) sessRun(tun tunnel.Tunnel) {
	defer func() {
		err := dialer.Remove(tun)
		if err != nil {
			logger.Error(err.Error())
		}
	}()

	tun.Loop()
	logger.Info("session runtime quit.")
	return
}

func (dialer *Dialer) Dial(network, address string) (net.Conn, error) {
	tun, err := dialer.Get()
	if err != nil {
		return nil, err
	}
	d, ok := tun.(netutil.Dialer)
	if !ok {
		panic("tunnel not a dialer in client side.")
	}
	return d.Dial(network, address)
}
