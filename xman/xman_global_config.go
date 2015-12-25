// copyright xman
// author    reezhou
// email     reechou@gmail.com
// file      xman_global_config.go

package xman

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"

	"config"
	. "logs"
)

var (
	ErrNoServer      = errors.New("No Server in config.")
	ErrSrvSection    = errors.New("Server section error.")
	ErrSrvStart      = errors.New("Server start error.")
	ErrMapNotFound   = errors.New("Map not found.")
	ErrUnknowNetwork = errors.New("Unknow network.")
)

type ReloadHandler func(config *xmanconfig.Configuration)

type MapVal struct {
	src     interface{}
	srcType int32
}

type XMANGlobalConfig struct {
	srvNum  int
	connNum int
	srvMap  map[string]*MapVal
	connMap map[string]*MapVal

	httpAddr      string
	reloadHandler ReloadHandler

	config *xmanconfig.Configuration
}

type ServerInfo struct {
	network   string
	address   string
	maxReq    int
	connProxy string
}

type ConnInfo struct {
	network string
	address string
	maxReq  int
	CMD     int32
}

var XMANConfig XMANGlobalConfig

func ParseToINT(src string) (int, error) {
	val, err := strconv.Atoi(src)
	if err != nil {
		Log(LOG_ERROR, "GlobalConfig: parse to int error, srcString[", src, "]")
		return 0, err
	}
	return val, err
}

func getServerInfo(s *xmanconfig.Section) *ServerInfo {
	srvInfo := &ServerInfo{}
	srvInfo.network = s.ValueOf("Network")
	srvInfo.address = s.ValueOf("Address")
	srvInfo.maxReq, _ = strconv.Atoi(s.ValueOf("MaxReq"))
	srvInfo.connProxy = s.ValueOf("ConnProxy")
	return srvInfo
}

func getConnInfo(s *xmanconfig.Section) *ConnInfo {
	connInfo := &ConnInfo{}
	connInfo.network = s.ValueOf("Network")
	connInfo.address = s.ValueOf("Address")
	connInfo.maxReq, _ = strconv.Atoi(s.ValueOf("MaxReq"))
	cmd, _ := strconv.Atoi(s.ValueOf("CMD"))
	connInfo.CMD = int32(cmd)
	return connInfo
}

func InitFromConfig(config *xmanconfig.Configuration) error {
	XMANConfig.config = config

	srvConfigSession, err := config.Section("ServerConfig")
	if err != nil {
		Log(LOG_ERROR, "GlobalConfig: can not find section[ServerConfig]")
		return err
	}
	srvNum, err := ParseToINT(srvConfigSession.ValueOf("SrvNum"))
	if err != nil {
		return err
	}
	XMANConfig.srvNum = srvNum
	if XMANConfig.srvNum == 0 {
		return ErrNoServer
	}
	connNum, err := ParseToINT(srvConfigSession.ValueOf("ConnNum"))
	XMANConfig.connNum = connNum
	XMANConfig.httpAddr = srvConfigSession.ValueOf("HttpAddr")

	// init server
	for i := 1; i <= XMANConfig.srvNum; i++ {
		srvSectionStr := "Server" + strconv.Itoa(i)
		srvSection, err := config.Section(srvSectionStr)
		if err != nil {
			Log(LOG_ERROR, "GlobalConfig: section error ->", srvSectionStr)
			return ErrSrvSection
		}
		srvInfo := getServerInfo(srvSection)
		if srvInfo.network == "tcp" {
			tcpServer, err := RegisterTCPServer(srvInfo.network, srvInfo.address, nil, nil, srvInfo.maxReq)
			if err != nil {
				Log(LOG_ERROR, "tcpserver:", srvInfo.address, "RegisterTCPServer error.")
				return ErrSrvStart
			}
			mapval := &MapVal{
				src:     tcpServer,
				srcType: TCP,
			}
			XMANConfig.srvMap[srvSectionStr] = mapval
		} else if srvInfo.network == "udp" {
			udpServer, err := RegisterUDPServer(srvInfo.network, srvInfo.address)
			if err != nil {
				Log(LOG_ERROR, "udpserver:", srvInfo.address, "RegisterUDPServer error.")
				return ErrSrvStart
			}
			mapval := &MapVal{
				src:     udpServer,
				srcType: UDP,
			}
			XMANConfig.srvMap[srvSectionStr] = mapval
		} else {
			return ErrUnknowNetwork
		}
	}

	// init conn
	for i := 1; i <= XMANConfig.connNum; i++ {
		connSectionStr := "Conn" + strconv.Itoa(i)
		connSection, err := config.Section(connSectionStr)
		if err != nil {
			Log(LOG_ERROR, "GlobalConfig: section error ->", connSectionStr)
			return ErrSrvSection
		}
		connInfo := getConnInfo(connSection)
		if connInfo.CMD == 0 {
			Log(LOG_ERROR, "connInfo error, CMD==0.", connSectionStr)
			continue
		}
		if connInfo.network == "tcp" {
			tcpConn := RegisterTCPConn(connInfo.CMD, connInfo.network, connInfo.address, connInfo.maxReq, nil, nil)
			mapval := &MapVal{
				src:     tcpConn,
				srcType: TCP,
			}
			XMANConfig.connMap[connSectionStr] = mapval
		}
	}

	fmt.Println(XMANConfig)

	return nil
}

func SetSrvHandler(name string, handlers interface{}) error {
	src, ok := XMANConfig.srvMap[name]
	if !ok {
		Log(LOG_ERROR, "GlobalConfig: SetSrvHandler map not found:", name)
		return ErrMapNotFound
	}
	if src.srcType == TCP {
		Log(LOG_DEBUG, "server: src.srcType == TCP", name)
		srcTCP := src.src.(*TCPServer)
		hs := handlers.(*TCPServerHandlers)
		srcTCP.SetProtoUnpackHandler(hs.UnpackHandler)
		srcTCP.SetPackageHandler(hs.PackageHandler)
	} else if src.srcType == UDP {
		Log(LOG_DEBUG, "server: src.srcType == UDP", name)
		srcUDP := src.src.(*UDPServer)
		hs := handlers.(*UDPServerHandlers)
		srcUDP.SetUnpackHandler(hs.UnpackHandler)
		srcUDP.SetPackageHandler(hs.PackageHandler)
	}
	return nil
}

func SetConnHandler(name string, handlers interface{}) error {
	src, ok := XMANConfig.connMap[name]
	if !ok {
		Log(LOG_ERROR, "GlobalConfig: SetConnHandler map not found.", name)
		return ErrMapNotFound
	}
	if src.srcType == TCP {
		Log(LOG_DEBUG, "conn: src.srcType == TCP", name)
		srcTCP := src.src.(*TCPConnMgr)
		hs := handlers.(*TCPConnHandlers)
		srcTCP.SetPackHandler(hs.PackHandler)
		srcTCP.SetUnpackHandler(hs.UnpackHandler)
	} else if src.srcType == UDP {
		Log(LOG_DEBUG, "conn: src.srcType == TCP", name)
		srcUDP := src.src.(*UDPConnMgr)
		hs := handlers.(*UDPConnHandlers)
		srcUDP.SetPackHandler(hs.PackHandler)
		srcUDP.SetUnpackHandler(hs.UnpackHandler)
	}
	return nil
}

func SetReloadHandler(handler ReloadHandler) {
	XMANConfig.reloadHandler = handler
}

func RunServer() {
	var wg sync.WaitGroup
	for k, v := range XMANConfig.srvMap {
		if v.srcType == TCP {
			Log(LOG_DEBUG, k, "TCP server start...")
			srvTCP := v.src.(*TCPServer)
			wg.Add(1)
			go srvTCP.AcceptLoop()
		} else if v.srcType == UDP {
			Log(LOG_DEBUG, k, "UDP server start...")
			srvUDP := v.src.(*UDPServer)
			wg.Add(1)
			go srvUDP.RunUDPServer()
		}
	}

	// init http server
	wg.Add(1)
	go StartHTTPSrv()

	wg.Wait()
}

func Reload(w http.ResponseWriter, req *http.Request) {
	var echoStr string
	if XMANConfig.reloadHandler == nil {
		echoStr = "ReloadHandler is nil, cannot reload config."
	} else {
		config, err := xmanconfig.Read(XMANConfig.config.FilePath())
		if err != nil {
			echoStr = "config read error, " + XMANConfig.config.FilePath()
		} else {
			XMANConfig.reloadHandler(config)
			echoStr = "Reload config success."
		}
	}
	io.WriteString(w, echoStr)
}

func StartHTTPSrv() {
	http.HandleFunc("/reload", Reload)
	if XMANConfig.httpAddr == "" {
		XMANConfig.httpAddr = ":8080"
	}
	err := http.ListenAndServe(XMANConfig.httpAddr, nil)
	if err != nil {
		Log(LOG_ERROR, "http listen error.", XMANConfig.httpAddr)
	}
}

func init() {
	XMANConfig.srvNum = 0
	XMANConfig.connNum = 0
	XMANConfig.srvMap = make(map[string]*MapVal)
	XMANConfig.connMap = make(map[string]*MapVal)
}
