// copyright xman
// author    reezhou
// email     reechou@gmail.com
// file      xman_conn_proxy.go

package xman

import (
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	. "logs"
)

var (
	ErrNotFountRemoteSrv  = errors.New("Remote SRV not found.")
	ErrRemoteSrvTimeout   = errors.New("Remote SRV timeout.")
	ErrRemoteSrvUnConnect = errors.New("Remote SRV not connected.")
	ErrConnHandlerNotInit = errors.New("Conn Handlers not init.")
	ErrTCPConnPoolGet     = errors.New("TCP Conn Pool has no alive connMgr.")
)

const (
	SRV_STATUS_OK        = 1
	SRV_STATUS_RECONNECT = 2
)

type Request struct {
	SrvTag   int32
	IsCancel bool
	ReqSeq   uint32
	ReqPkg   []byte
	RspChan  chan interface{}
}

type SrvInfo struct {
	srvType      int32
	srvStatus    int32
	packHandler  ConnPackHandler
	remoteSrvSeq uint32
	connMgr      interface{}
}

type RemoteSrvInfo struct {
	sync.Mutex
	srvIdx    int
	allSrvNum int
	srvInfos  [maxRemoteSrvIP]*SrvInfo
}

type ConnPackHandler func(protoPkg interface{}, seq uint32) []byte
type UDPConnUnpackHandler func(buf []byte) (uint32, interface{}, error)
type TCPConnUnpackHandler func(pkgBuf []byte, seqMap *map[uint32]interface{}) []byte

var RemoteSrvChanList = make(map[int32]*RemoteSrvInfo)

// remote srv request
// srvTag: maybe CMD
func RequestToRemote(srvTag int32, req interface{}, timeout time.Duration) (interface{}, error) {
	remoteSrv, err := chooseRemoteSrv(srvTag)
	if err != nil {
		Log(LOG_ERROR, "chooseRemoteSrv err.", err)
		return nil, err
	}
	newSeq := atomic.AddUint32(&(remoteSrv.remoteSrvSeq), 1)
	if remoteSrv.packHandler == nil {
		Log(LOG_ERROR, ErrConnHandlerNotInit)
		panic(ErrConnHandlerNotInit)
	}
	reqBuf := remoteSrv.packHandler(req, newSeq)
	reqPkg := &Request{
		IsCancel: false,
		ReqSeq:   newSeq,
		ReqPkg:   reqBuf,
		RspChan:  make(chan interface{}, 1),
	}
	var valChan chan *Request
	if remoteSrv.srvType == UDP {
		valChan = remoteSrv.connMgr.(*UDPConnMgr).GetReqChan()
	} else {
		valChan = remoteSrv.connMgr.(*TCPConnMgr).GetReqChan()
	}
	valChan <- reqPkg

	select {
	case rspPkg := <-reqPkg.RspChan:
		return rspPkg, nil
	case <-time.After(timeout * time.Second):
		Log(LOG_ERROR, "seq:", newSeq, "Remote srv timeout")
		valChan <- &Request{IsCancel: true, ReqSeq: newSeq}
		return nil, ErrRemoteSrvTimeout
	}
}

func chooseRemoteSrv(srvTag int32) (*SrvInfo, error) {
	remoteSrvList, ok := RemoteSrvChanList[srvTag]
	if !ok {
		Log(LOG_ERROR, "RemoteSrvChanList has no srvTag:", srvTag)
		return nil, ErrNotFountRemoteSrv
	}
	addSrvIdx(remoteSrvList)
	remoteSrv := remoteSrvList.srvInfos[remoteSrvList.srvIdx]
	if remoteSrv.srvStatus != SRV_STATUS_OK {
		Log(LOG_ERROR, "RemoteSrvChanList srv1 status not ok. srvIdx:", remoteSrvList.srvIdx, "status:", remoteSrv.srvStatus)
		if checkTcpReconnect(remoteSrv.connMgr.(*TCPConnMgr)) {
			return remoteSrv, nil
		}
		addSrvIdx(remoteSrvList)
		remoteSrv2 := remoteSrvList.srvInfos[remoteSrvList.srvIdx]
		if remoteSrv2.srvStatus != SRV_STATUS_OK {
			Log(LOG_ERROR, "RemoteSrvChanList srv2 status not ok. srvIdx:", remoteSrvList.srvIdx, "status:", remoteSrv.srvStatus)
			if checkTcpReconnect(remoteSrv2.connMgr.(*TCPConnMgr)) {
				return remoteSrv2, nil
			} else {
				return nil, ErrRemoteSrvUnConnect
			}
		} else {
			return remoteSrv2, nil
		}
	} else {
		return remoteSrv, nil
	}
}

func checkTcpReconnect(tcpMgr *TCPConnMgr) bool {
	nowTime := time.Now().Unix()
	if nowTime-tcpMgr.closeConnTime > tcpConnCheckReconnect {
		return tcpMgr.tcpConnReconnect()
	}
	return false
}

func addSrvIdx(remoteSrvList *RemoteSrvInfo) {
	remoteSrvList.Lock()
	defer remoteSrvList.Unlock()
	remoteSrvList.srvIdx = (remoteSrvList.srvIdx + 1) % remoteSrvList.allSrvNum
}

// ------ UDP Conn ------
type UDPConnMgr struct {
	conn             *net.UDPConn
	reqChan          chan *Request
	sendChan         chan []byte
	recvChan         chan []byte
	reqMap           map[uint32]*Request
	udpUnpackHandler UDPConnUnpackHandler
	srvInfo          *SrvInfo
}

type UDPConnHandlers struct {
	PackHandler   ConnPackHandler
	UnpackHandler UDPConnUnpackHandler
}

func RegisterUDPConn(srvTag int32, network, addr string, reqMaxSize int, udpPackHandler ConnPackHandler, udpUnpackHandler UDPConnUnpackHandler) {
	//	if udpUnpackHandler == nil {
	//		panic("xman_conn_proxy: RegisterUDPConn srvHandler is nil")
	//	}
	srvInfo := &SrvInfo{
		srvType:      UDP,
		srvStatus:    SRV_STATUS_OK,
		packHandler:  udpPackHandler,
		remoteSrvSeq: 0,
	}
	if remoteSrvList, ok := RemoteSrvChanList[srvTag]; !ok {
		remoteSrvInfo := &RemoteSrvInfo{}
		remoteSrvInfo.srvIdx = 0
		remoteSrvInfo.allSrvNum = 0
		remoteSrvInfo.srvInfos[remoteSrvInfo.allSrvNum] = srvInfo
		remoteSrvInfo.allSrvNum++
		RemoteSrvChanList[srvTag] = remoteSrvInfo
	} else {
		if remoteSrvList.allSrvNum >= maxRemoteSrvIP {
			panic("xman_conn_proxy: RegisterUDPConn remoteSrvList.allSrvNum >= maxRemoteSrvIP")
			return
		}
		remoteSrvList.srvInfos[remoteSrvList.allSrvNum] = srvInfo
		remoteSrvList.allSrvNum++
	}
	reqChan := make(chan *Request, reqMaxSize)
	udpConnMgr := UDPConnHandler(reqChan, network, addr, reqMaxSize, udpUnpackHandler)
	udpConnMgr.srvInfo = srvInfo
	srvInfo.connMgr = udpConnMgr
	go udpConnMgr.udpConnStart()
}

func UDPConnHandler(reqChan chan *Request, network, addr string, reqMaxSize int, udpUnpackHandler UDPConnUnpackHandler) *UDPConnMgr {
	serviceAddr, err := net.ResolveUDPAddr(network, addr)
	if err != nil {
		Log(LOG_ERROR, "net.ResolveUDPAddr failed.", err)
		os.Exit(1)
	}

	conn, err := net.DialUDP(network, nil, serviceAddr)
	if err != nil {
		Log(LOG_ERROR, "net.DialUDP failed.", err)
		os.Exit(1)
	}

	return &UDPConnMgr{
		conn:             conn,
		reqChan:          reqChan,
		sendChan:         make(chan []byte, reqMaxSize),
		recvChan:         make(chan []byte, reqMaxSize),
		reqMap:           make(map[uint32]*Request),
		udpUnpackHandler: udpUnpackHandler,
	}
}

func (udpMgr *UDPConnMgr) GetReqChan() chan *Request {
	return udpMgr.reqChan
}

func (udpMgr *UDPConnMgr) SetPackHandler(udpPackHandler ConnPackHandler) {
	udpMgr.srvInfo.packHandler = udpPackHandler
}

func (udpMgr *UDPConnMgr) SetUnpackHandler(udpUnpackHandler UDPConnUnpackHandler) {
	udpMgr.udpUnpackHandler = udpUnpackHandler
}

func (udpMgr *UDPConnMgr) udpConnStart() {
	defer udpMgr.conn.Close()

	go udpMgr.udpSendHandler()
	go udpMgr.udpRecvHandler()

	for {
		select {
		case req := <-udpMgr.reqChan:
			if req.IsCancel {
				delete(udpMgr.reqMap, req.ReqSeq)
				continue
			}
			udpMgr.reqMap[req.ReqSeq] = req
			udpMgr.sendChan <- req.ReqPkg
			Log(LOG_NETWORK, "NormalRequest recv, reqSeq ", req.ReqSeq)
		case rsp := <-udpMgr.recvChan:
			seq, rspPkg, err := udpMgr.udpUnpackHandler(rsp)
			if err != nil {
				Log(LOG_ERROR, "udpMgr.udpUnpackHandler failed.", err)
				continue
			}
			req, ok := udpMgr.reqMap[seq]
			if !ok {
				Log(LOG_ERROR, "seq not found. seq:", seq)
				continue
			}
			req.RspChan <- rspPkg
			delete(udpMgr.reqMap, req.ReqSeq)
		}
	}
}

func (udpMgr *UDPConnMgr) udpSendHandler() {
	for data := range udpMgr.sendChan {
		wlen, err := udpMgr.conn.Write(data)
		if err != nil || wlen != len(data) {
			Log(LOG_ERROR, "conn.Write failed.", err)
			continue
		}
	}
}

func (udpMgr *UDPConnMgr) udpRecvHandler() {
	for {
		buf := make([]byte, recvChanBufLen)
		rlen, err := udpMgr.conn.Read(buf)
		if err != nil || rlen <= 0 {
			Log(LOG_ERROR, "conn.Read failed.", err)
			continue
		}
		udpMgr.recvChan <- buf[:rlen]
	}
}

// ------ TCP Conn ------
type TCPConnMgr struct {
	closed           int32
	srvStatus        int32
	network          string
	addr             string
	conn             *net.TCPConn
	pauseConn        chan bool
	reqChan          chan *Request
	sendChan         chan []byte
	recvChan         chan map[uint32]interface{}
	reqMap           map[uint32]*Request
	tcpUnpackHandler TCPConnUnpackHandler
	srvInfo          *SrvInfo
	reconnectTicker  *time.Ticker
	closeConnTime    int64
}

type TCPConnHandlers struct {
	PackHandler   ConnPackHandler
	UnpackHandler TCPConnUnpackHandler
}

func RegisterTCPConn(srvTag int32, network, addr string, reqMaxSize int, tcpPackHandler ConnPackHandler, tcpUnpackHandler TCPConnUnpackHandler) *TCPConnMgr {
	//	if tcpUnpackHandler == nil {
	//		panic("xman_conn_proxy: register srvHandler is nil")
	//	}
	srvInfo := &SrvInfo{
		srvType:      TCP,
		srvStatus:    SRV_STATUS_OK,
		packHandler:  tcpPackHandler,
		remoteSrvSeq: 0,
	}
	if remoteSrvList, ok := RemoteSrvChanList[srvTag]; !ok {
		remoteSrvInfo := &RemoteSrvInfo{}
		remoteSrvInfo.srvIdx = 0
		remoteSrvInfo.allSrvNum = 0
		remoteSrvInfo.srvInfos[remoteSrvInfo.allSrvNum] = srvInfo
		remoteSrvInfo.allSrvNum++
		RemoteSrvChanList[srvTag] = remoteSrvInfo
	} else {
		if remoteSrvList.allSrvNum >= maxRemoteSrvIP {
			panic("xman_conn_proxy: RegisterUDPConn remoteSrvList.allSrvNum >= maxRemoteSrvIP")
			return nil
		}
		remoteSrvList.srvInfos[remoteSrvList.allSrvNum] = srvInfo
		remoteSrvList.allSrvNum++
	}
	reqChan := make(chan *Request, reqMaxSize)
	tcpConnMgr := TCPConnHandler(reqChan, network, addr, reqMaxSize, tcpUnpackHandler)
	tcpConnMgr.srvInfo = srvInfo
	srvInfo.connMgr = tcpConnMgr
	Log(LOG_DEBUG, "tcp_conn: srvStatuc:", tcpConnMgr.srvInfo.srvStatus)
	go tcpConnMgr.tcpConnStart()

	return tcpConnMgr
}

func TCPConnHandler(reqChan chan *Request, network, addr string, reqMaxSize int, tcpUnpackHandler TCPConnUnpackHandler) *TCPConnMgr {
	tcpAddr, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		Log(LOG_ERROR, "tcp_conn: net.ResolveTCPAddr failed.", err)
		os.Exit(1)
	}
	conn, err := net.DialTCP(network, nil, tcpAddr)
	if err != nil {
		Log(LOG_ERROR, "tcp_conn: net.DialTCP failed.", err)
		os.Exit(1)
	}

	return &TCPConnMgr{
		closed:           -1,
		srvStatus:        SRV_STATUS_OK,
		network:          network,
		addr:             addr,
		conn:             conn,
		pauseConn:        make(chan bool),
		reqChan:          reqChan,
		sendChan:         make(chan []byte, reqMaxSize),
		recvChan:         make(chan map[uint32]interface{}, reqMaxSize),
		reqMap:           make(map[uint32]*Request),
		tcpUnpackHandler: tcpUnpackHandler,
	}
}

func (tcpMgr *TCPConnMgr) GetReqChan() chan *Request {
	return tcpMgr.reqChan
}

func (tcpMgr *TCPConnMgr) SetPackHandler(tcpPackHandler ConnPackHandler) {
	tcpMgr.srvInfo.packHandler = tcpPackHandler
}

func (tcpMgr *TCPConnMgr) SetUnpackHandler(tcpUnpackHandler TCPConnUnpackHandler) {
	tcpMgr.tcpUnpackHandler = tcpUnpackHandler
}

func (tcpMgr *TCPConnMgr) tcpConnStart() {
	//	defer tcpMgr.conn.Close()

	tcpMgr.tcpConnSendAndRecv()

	for {
		select {
		case req := <-tcpMgr.reqChan:
			if req.IsCancel {
				delete(tcpMgr.reqMap, req.ReqSeq)
				continue
			}
			tcpMgr.reqMap[req.ReqSeq] = req
			tcpMgr.sendChan <- req.ReqPkg
			Log(LOG_NETWORK, "tcp_conn: NormalRequest recv, reqSeq ", req.ReqSeq)
		case rsp := <-tcpMgr.recvChan:
			for seq, pkg := range rsp {
				req, ok := tcpMgr.reqMap[seq]
				if !ok {
					Log(LOG_ERROR, "tcp_conn: tcpConnMgr: seq not found. seq:", seq)
					continue
				}
				req.RspChan <- pkg
				delete(tcpMgr.reqMap, req.ReqSeq)
			}
		}
	}
}

func (tcpMgr *TCPConnMgr) tcpSendHandler() {
	defer tcpMgr.tcpConnClose()

	//	for data := range tcpMgr.sendChan {
	//		wlen, err := tcpMgr.conn.Write(data)
	//		if err != nil || wlen != len(data) {
	//			catchError("tcpSendHandler: conn.Write failed.", err)
	//			continue
	//		}
	//	}

	for {
		select {
		case data := <-tcpMgr.sendChan:
			wlen, err := tcpMgr.conn.Write(data)
			if err != nil || wlen != len(data) {
				Log(LOG_ERROR, "tcp_conn: tcpSendHandler: conn.Write failed.", err)
				continue
			}
		case <-tcpMgr.pauseConn:
			Log(LOG_ERROR, "tcp_conn: tcpSendHandler recieve conn pause.")
			return
		}
	}
}

func (tcpMgr *TCPConnMgr) tcpRecvHandler() {
	defer tcpMgr.tcpConnClose()

	tmpbuf := make([]byte, 0)
	for {
		buf := make([]byte, recvChanBufLen)
		rlen, err := tcpMgr.conn.Read(buf)
		if err != nil || rlen <= 0 {
			if err == io.EOF {
				Log(LOG_ERROR, "tcp_conn: remote srv closed.")
				return
			}
			Log(LOG_ERROR, "tcp_conn: tcpRecvHandler: conn.Read failed.", err)
			return
		}
		seqMap := make(map[uint32]interface{})
		if tcpMgr.tcpUnpackHandler == nil {
			panic(ErrConnHandlerNotInit)
		}
		tmpbuf = tcpMgr.tcpUnpackHandler(append(tmpbuf, buf[:rlen]...), &seqMap)
		if len(seqMap) > 0 {
			tcpMgr.recvChan <- seqMap
		}
	}
}

func (tcpMgr *TCPConnMgr) tcpConnClose() {
	if atomic.CompareAndSwapInt32(&tcpMgr.closed, 0, -1) {
		tcpMgr.conn.Close()
		close(tcpMgr.pauseConn)
		//		tcpMgr.srvInfo.srvStatus = SRV_STATUS_RECONNECT
		tcpMgr.srvStatus = SRV_STATUS_RECONNECT
		tcpMgr.closeConnTime = time.Now().Unix()
	}
}

func (tcpMgr *TCPConnMgr) tcpConnSendAndRecv() {
	if atomic.CompareAndSwapInt32(&tcpMgr.closed, -1, 0) {
		go tcpMgr.tcpSendHandler()
		go tcpMgr.tcpRecvHandler()
	}
}

func (tcpMgr *TCPConnMgr) tcpConnReconnect() bool {
	Log(LOG_DEBUG, "tcp_conn: tcp conn reconnect ->", tcpMgr.addr)
	tcpAddr, err := net.ResolveTCPAddr(tcpMgr.network, tcpMgr.addr)
	if err != nil {
		Log(LOG_ERROR, "tcp_conn: net.ResolveTCPAddr failed.", err)
		return false
	}

	conn, err := net.DialTCP(tcpMgr.network, nil, tcpAddr)
	if err != nil {
		Log(LOG_ERROR, "tcp_conn: net.DialTCP failed.", err)
		return false
	}

	Log(LOG_DEFAULT, "tcp_conn: reconnect success.", tcpMgr.addr)
	tcpMgr.conn = conn
	tcpMgr.srvInfo.srvStatus = SRV_STATUS_OK
	tcpMgr.pauseConn = make(chan bool)
	tcpMgr.tcpConnSendAndRecv()

	return true
}

func (tcpMgr *TCPConnMgr) tcpConnReconnectTickerStart() {
	tcpMgr.reconnectTicker = time.NewTicker(10 * time.Second)
	Log(LOG_DEBUG, "tcp_conn: tcpMgr reconnectTicker start...")
	for t := range tcpMgr.reconnectTicker.C {
		if tcpMgr.tcpConnReconnect() {
			tcpMgr.reconnectTicker.Stop()
			tcpMgr.tcpConnSendAndRecv()
			Log(LOG_DEBUG, "tcp_conn: tcpMgr reconnectTicker stopped. and tcp conn send and recv restart...", t)

			break
		}
	}
}

func (tcpMgr *TCPConnMgr) tcpConnPause() {
	Log(LOG_ERROR, "tcp_conn: tcpConnPause...")
	tcpMgr.pauseConn <- true
	go tcpMgr.tcpConnReconnectTickerStart()
}

// ------ TCP Conn Pool ------
type TCPConnPool struct {
	maxIdle int
	conns   chan *TCPConnMgr
	srvInfo *SrvInfo
}

func NewTCPConnPool(maxIdle int, network, addr string, reqMaxSize int, tcpUnpackHandler TCPConnUnpackHandler) *TCPConnPool {
	pool := &TCPConnPool{maxIdle: maxIdle, conns: make(chan *TCPConnMgr, maxIdle)}
	for i := 0; i < maxIdle; i++ {
		reqChan := make(chan *Request, reqMaxSize)
		tcpConnMgr := TCPConnHandler(reqChan, network, addr, reqMaxSize, tcpUnpackHandler)
		pool.conns <- tcpConnMgr
		go tcpConnMgr.tcpConnStart()
	}
	return pool
}

func (pool *TCPConnPool) Get() (*TCPConnMgr, error) {
	i := 0
	for tcpConnMgr := range pool.conns {
		defer pool.Release(tcpConnMgr)
		if tcpConnMgr.srvStatus == SRV_STATUS_OK {
			return tcpConnMgr, nil
		}
		i++
		if i >= pool.maxIdle/2 {
			Log(LOG_ERROR, "i >= pool.maxIdle/2", i)
			pool.srvInfo.srvStatus = SRV_STATUS_RECONNECT
			break
		}
	}
	return nil, ErrTCPConnPoolGet
}

func (pool *TCPConnPool) Release(tcpConnMgr *TCPConnMgr) {
	pool.conns <- tcpConnMgr
}

func (pool *TCPConnPool) Reconnect() *TCPConnMgr {
	i := 0
	for tcpConnMgr := range pool.conns {
		defer pool.Release(tcpConnMgr)
		if tcpConnMgr.srvStatus == SRV_STATUS_OK {
			return tcpConnMgr
		}
		if tcpConnMgr.tcpConnReconnect() {
			tcpConnMgr.srvStatus = SRV_STATUS_OK
			return tcpConnMgr
		}
		i++
		if i >= pool.maxIdle/2 {
			break
		}
	}
	return nil
}
