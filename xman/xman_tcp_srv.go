// copyright xman
// author    reezhou
// email     reechou@gmail.com
// file      xman_tcp_srv.go

package xman

import (
	"errors"
	"net"
	"time"

	. "logs"
	"utils"
)

var (
	ErrTCPServerHandlerNotInit = errors.New("TCP Server handlers not init.")
)

type TCPServer struct {
	listener     net.Listener
	sendChanSize int
	//	nowSeq uint32
	protoUnpackHandler ProtoUnpackHandler
	packageHandler     PackageHandler

	tw *utils.TimingWheel
}

func RegisterTCPServer(network, addr string, protoUnpackHandler ProtoUnpackHandler, packageHandler PackageHandler, sendChanSize int) (*TCPServer, error) {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	return NewTCPServer(listener, protoUnpackHandler, packageHandler, sendChanSize), nil
}

func NewTCPServer(listener net.Listener, protoUnpackHandler ProtoUnpackHandler, packageHandler PackageHandler, sendChanSize int) *TCPServer {
	return &TCPServer{
		listener:     listener,
		sendChanSize: sendChanSize,
		//		nowSeq: 0,
		protoUnpackHandler: protoUnpackHandler,
		packageHandler:     packageHandler,
		tw:                 utils.NewTimingWheel(1*time.Second, 3600),
	}
}

func (s *TCPServer) SetProtoUnpackHandler(protoUnpackHandler ProtoUnpackHandler) {
	s.protoUnpackHandler = protoUnpackHandler
}

func (s *TCPServer) SetPackageHandler(packageHandler PackageHandler) {
	s.packageHandler = packageHandler
}

func (s *TCPServer) Close() error {
	return s.listener.Close()
}

func (s *TCPServer) AcceptLoop() error {
	if s.protoUnpackHandler == nil || s.packageHandler == nil {
		Log(LOG_ERROR, ErrTCPServerHandlerNotInit)
		panic(ErrTCPServerHandlerNotInit)
	}

	defer s.Close()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				continue
			} else {
				Log(LOG_ERROR, "tcpserver listener accept error.", err)
				return err
			}
		}
		session := NewSession(conn, s.protoUnpackHandler, s.packageHandler, s.sendChanSize, s.tw)
		//		s.nowSeq = (s.nowSeq + maxSessionSeq) % maxUINT32
		tcpConn := session.RawConn().(*net.TCPConn)
		tcpConn.SetNoDelay(true)
		tcpConn.SetReadBuffer(tcpReadBufLen)
		tcpConn.SetWriteBuffer(tcpWriteBufLen)

		session.Start()
	}
}
