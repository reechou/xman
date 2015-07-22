// copyright xman
// author    reezhou
// email     reechou@gmail.com
// file      xman_session.go

package xman

import (
	"errors"
	"net"
	"sync/atomic"
	"io"
	"time"

	. "logs"
	"utils"
)

var (
	ErrStopped = errors.New("xman: TCPSession stopped")
	ErrSendChanFull = errors.New("xman: send chan full")
)

type ProtoUnpackHandler func(pkgBuf []byte, pkgChan chan interface{}) []byte
type PackageHandler func(s *TCPSession, pkg interface{})

type TCPServerHandlers struct {
	UnpackHandler ProtoUnpackHandler
	PackageHandler PackageHandler
}

type TCPSession struct {
	closed int32
	conn net.Conn
	stopSessionChan chan bool
//	recvChan chan []byte
	pkgChan chan interface{}
	sendChan chan []byte
//	startSeq uint32
//	maxSeq uint32
//	nowSeq uint32
	protoUnpackHandler ProtoUnpackHandler
	packageHandler PackageHandler

	lastRecvUnixTime int64
	tw *utils.TimingWheel
}

func NewSession(conn net.Conn, protoUnpackHandler ProtoUnpackHandler, packageHandler PackageHandler, sendChanSize int, tw *utils.TimingWheel) *TCPSession {
	return &TCPSession{
		closed: -1,
		conn: conn,
		stopSessionChan: make(chan bool),
		pkgChan: make(chan interface{}, sendChanSize),
		sendChan: make(chan []byte, sendChanSize),
		protoUnpackHandler: protoUnpackHandler,
		lastRecvUnixTime: time.Now().Unix(),
		packageHandler: packageHandler,
		tw: tw,
	}
}

func (s *TCPSession) RawConn() net.Conn {
	return s.conn
}

func (s *TCPSession) Close() {
	if atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		Log(LOG_DEBUG, "TCPSession closed.", s.conn.RemoteAddr().String())
		s.conn.Close()
		close(s.stopSessionChan)
	}
}

func (s *TCPSession) Start() {
	if atomic.CompareAndSwapInt32(&s.closed, -1, 0) {
		go s.recvLoop()
		go s.sendLoop()
		go s.pkgChanLoop()
	}
}

func (s *TCPSession) recvLoop() {
	defer s.Close()

	tmpbuf := make([]byte, 0)
	buf := make([]byte, RecvReqBufLen)
	for {
		n, err := s.conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				Log(LOG_DEBUG, "client closed.")
				break
			}
			Log(LOG_ERROR, "tcp connection read error.", err)
			break
		}
		s.lastRecvUnixTime = time.Now().Unix()
		tmpbuf = s.protoUnpackHandler(append(tmpbuf, buf[:n]...), s.pkgChan)
	}
}

func (s *TCPSession) sendLoop() {
	defer s.Close()

	var sendbuf []byte
	for {
		select {
		case sendbuf = <- s.sendChan:
			_, err := s.conn.Write(sendbuf)
			if err != nil {
				Log(LOG_ERROR, "tcp connection write error.", err)
				return
			}
		case <- s.tw.After(sessionCheck * time.Second):
			nowTime := time.Now().Unix()
			if nowTime - s.lastRecvUnixTime > sessionTimeout {
				Log(LOG_ERROR, "sessionTimeout, close TCPSession.")
				return
			}
		case <- s.stopSessionChan:
			return
		}
	}
}

func (s *TCPSession) pkgChanLoop() {
//	for pkg := range s.pkgChan {
//		seq := s.nowSeq
//		s.nowSeq = (s.nowSeq + 1) % s.maxSeq + s.startSeq
//		go s.packageHandler(s, pkg, 0)
//	}

	for {
		select {
		case pkg := <-s.pkgChan:
			go s.packageHandler(s, pkg)
		case <- s.stopSessionChan:
			Log(LOG_DEBUG, "get msg of close stopChan.")
			return
		}
	}
}

func (s *TCPSession) AsyncSend(packageBuf []byte) error {
	select {
	case s.sendChan <- packageBuf:
	case <- s.stopSessionChan:
		return ErrStopped
	default:
		return ErrSendChanFull
	}
	return nil
}
