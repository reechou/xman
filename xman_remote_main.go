package main

import (
	"fmt"
	"strconv"

	"config"
	. "logs"
	. "xman"
	"demo"
)

const (
	REMOTE_SRV_START = iota
	TEST_SRV
	REMOTE_SRV_END
)

const (
	UDPSRV_START = iota
	UDPTESTSRV
	UDPSRV_END
)

type RemoteUDPSrv struct {
}

func (udpsrv *RemoteUDPSrv) ParseFromRecvBUf(buf []byte) (uint32, error) {
	Log(LOG_DEBUG, "in RemoteUDPSrv ParseFromRecvBUF")
	seq, err := strconv.Atoi(string(buf))
	return uint32(seq), err
}

type SrvProcessLogic struct {
}

func (logic *SrvProcessLogic) GoProcessLogic(msg []byte, seq uint32) {
	Log(LOG_DEBUG, "in GoProcessLogic. req:", string(msg))
	//	rspChan := make(chan []byte, 1)
//	reqChan := GetRemoteSrvChan(TEST_SRV)
//	reqChan <- &Request{false, seq, []byte(strconv.Itoa(int(seq))), rspChan}
//	var rsp []byte
//	ret := SelectChan(&rsp, rspChan, 2, reqChan, seq)
//	if ret != RET_OK {
//		SendToClientUDP(UDPTESTSRV, []byte("wait for rsp timeout."), seq)
//		return
//	}
//	SendToClientUDP(UDPTESTSRV, msg, seq)
}

func RemoteTCPLogicHandler(s *TCPSession, pkg interface{}) {
	realPkg := pkg.(*xmandemo.ProtoPkg)
	Log(LOG_DEBUG, "cmd:", realPkg.Header.Cmd, "seq:", realPkg.Header.Seq, "pkg_body:", realPkg.Body)
	pkgData := xmandemo.Pack(realPkg)
	s.AsyncSend(pkgData)
}

func main() {
	fmt.Println("XMAN Test")
	config, err := xmanconfig.Read("/Users/reezhou/Desktop/xman/src/config/srv.ini")
	if err != nil {
		fmt.Println(err)
		return
	}
	section, err := config.Section("MysqlInfo")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(section.ValueOf("host"))

	SetConsoleShow(true)
//	SetRollingDaily("/Users/reezhou/Desktop/xman/src/logs", "test.log")
	SetRollingFile("/Users/reezhou/Desktop/xman/src/logs", "rolling.log", 10, 50, MB)
	SetLogLevel(LOG_DEBUG)
//	Log(LOG_ERROR, "uin error")

	// RegisterUDPServer maybe need before RegisterUDPConn
	// Prevent port is occupied
//	RegisterUDPServer(UDPTESTSRV, ":6001")
	// Run server loop
//	RunUDPServer(UDPTESTSRV, &SrvProcessLogic{})

	// TCP Server
	tcpServer, err := RegisterTCPServer("tcp", ":7001", xmandemo.UnpackTCP, RemoteTCPLogicHandler, MaxReqChanLen)
	if err != nil {
		Log(LOG_ERROR, "tcp server listen err.", err)
		return
	}
	defer tcpServer.Close()

	tcpServer.AcceptLoop()
}
