// copyright xman
// author    reezhou
// email     reechou@gmail.com
// file      xman_udp_srv.go

package xman

import (
	"errors"
	"net"

	. "logs"
)

//var UDPConns [maxSrvNum]*net.UDPConn
//var UDPRemoteSessions [maxSrvNum]map[uint32]*net.UDPAddr

var (
	ErrUDPServerHandlerNotInit = errors.New("UDP server handlers not init.")
)

type UDPSrvUnpackHandler func(pkgBuf []byte) (interface{}, error)
type UDPSrvPackageHandler func(session *UDPSession, pkg interface{})

type UDPServerHandlers struct {
	UnpackHandler UDPSrvUnpackHandler
	PackageHandler UDPSrvPackageHandler
}

type UDPServer struct {
	addr string
	conn *net.UDPConn
	unpackHandler UDPSrvUnpackHandler
	packageHandler UDPSrvPackageHandler
}

type UDPSession struct {
	udpServer *UDPServer
	remoteAddr *net.UDPAddr
}

func RegisterUDPServer(network, addrStr string) (*UDPServer, error) {
	addr, err := net.ResolveUDPAddr(network, addrStr)
	if err != nil {
		Log(LOG_ERROR, "net.ResolveUDPAddr failed.", err)
		return nil, err
	}

	conn, err := net.ListenUDP(network, addr)
	if err != nil {
		Log(LOG_ERROR, "net.ListenUDP failed.", err)
		return nil, err
	}

	Log(LOG_DEBUG, "RegisterUDPServer success,", addr)

	return &UDPServer{
		addr: addrStr,
		conn: conn,
	}, nil
}

func (us *UDPServer) Close() {
	us.conn.Close()
}

func (us *UDPServer) SetUnpackHandler(unpackHandler UDPSrvUnpackHandler) {
	us.unpackHandler = unpackHandler
}

func (us *UDPServer) SetPackageHandler(packageHandler UDPSrvPackageHandler) {
	us.packageHandler = packageHandler
}

func (us *UDPServer) RunUDPServer() {
	if us.unpackHandler == nil || us.packageHandler == nil {
		panic(ErrUDPServerHandlerNotInit)
	}
	defer us.Close()

	for {
		buf := make([]byte, RecvReqBufLen)
		rlen, remote, err := us.conn.ReadFromUDP(buf)
		if err != nil {
			Log(LOG_ERROR, "conn.ReadFromUDP failed.", us.addr, err)
			continue
		}
		pkg, err := us.unpackHandler(buf[:rlen])
		if err != nil {
			Log(LOG_ERROR, "us.unpackHandler error.", us.addr, err)
			continue
		}
		go us.packageHandler(&UDPSession{udpServer: us, remoteAddr: remote}, pkg)
	}
}

func (us *UDPSession) SendToClientUDP(msg []byte) {
	us.udpServer.conn.WriteToUDP(msg, us.remoteAddr)
}
