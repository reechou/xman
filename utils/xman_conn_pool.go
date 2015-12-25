// copyright xman
// author    reezhou
// email     reechou@gmail.com
// file      xman_conn_pool.go

package utils

import (
	"time"
)

var nowFunc = time.Now

type ConnPool struct {
	Dial      func() (interface{}, error)
	MaxIdle   int
	MaxActive int
	closed    int
	active    int

	idle chan interface{}
}

type idleConn struct {
	c interface{}
	t time.Time
}

func (cp *ConnPool) InitPool() error {
	cp.idle = make(chan interface{}, cp.MaxActive)
	for i := 0; i < cp.MaxActive; i++ {
		connCtrl, err := cp.Dial()
		if err != nil {
			return err
		}
		cp.idle <- idleConn{t: nowFunc(), c: connCtrl}
	}
	return nil
}

func (cp *ConnPool) Get() interface{} {
	if cp.idle == nil {
		cp.InitPool()
	}

	ic := <-cp.idle
	connCtrl := ic.(idleConn).c
	defer cp.Release(connCtrl)

	return connCtrl
}

func (cp *ConnPool) Release(conn interface{}) {
	cp.idle <- idleConn{t: nowFunc(), c: conn}
}
