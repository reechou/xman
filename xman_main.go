package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"

	"github.com/golang/protobuf/proto"
	//	"github.com/garyburd/redigo/redis"

	"config"
	"db"
	"demo"
	. "logs"
	"time"
	. "xman"
)

const TESTSRV_CMD = 121

var redisController *xmandb.RedisController
var mysqlController *xmandb.MysqlController

func UDPLogicHandler(s *UDPSession, pkg interface{}) {
	realPkg := pkg.(*xmandemo.ProtoPkg)
	Log(LOG_DEBUG, "udp_logic: cmd:", realPkg.Header.Cmd, "pkg_body:", realPkg.Body)
	pkgData := xmandemo.Pack(realPkg)
	s.SendToClientUDP(pkgData)
}

func TCPLogicHandler(s *TCPSession, pkg interface{}) {
	realPkg := pkg.(*xmandemo.ProtoPkg)
	Log(LOG_DEBUG, "tcp_logic: cmd:", realPkg.Header.Cmd, "pkg_body:", realPkg.Body)
	//	realPkg.Header.Seq = seq
	//	pkgData := xmandemo.Pack(realPkg)
	//	remotePkg, err := RequestToRemote(TESTSRV_CMD, realPkg, 5)
	//	if err != nil {
	//		Log(LOG_ERROR, "RequestToRemote error.", err)
	//		return
	//	}
	//	realRemotePkg := remotePkg.(*xmandemo.ProtoPkg)
	//	Log(LOG_DEBUG, "remote cmd:", realRemotePkg.Header.Cmd, "seq:", realRemotePkg.Header.Seq, "remote_pkg_body:", realRemotePkg.Body)
	if rand.Intn(100) > 60 {
		return
	}
	time.Sleep(140 * time.Millisecond)
	retPkgData := xmandemo.Pack(realPkg)
	s.AsyncSend(retPkgData)
}

func InitConfig(path string) *xmanconfig.Configuration {
	config, err := xmanconfig.Read(path)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return config
}

func ReloadConfigure(config *xmanconfig.Configuration) {
	fmt.Println("in reloadconfig")
	section, _ := config.Section("MysqlInfo")
	fmt.Println(section.ValueOf("password"))
}

func goServer(config *xmanconfig.Configuration) {
	//	tcpServer, err := RegisterTCPServer("tcp", ":7000", xmandemo.UnPack, TCPLogicHandler, MaxReqChanLen)
	//	if err != nil {
	//		Log(LOG_ERROR, "tcp server listen err.", err)
	//		return
	//	}
	//	defer tcpServer.Close()
	//	RegisterTCPConn(TESTSRV_CMD, "tcp", ":7001", MaxReqChanLen, xmandemo.PackFromInterface, xmandemo.UnPackTCP)
	//	tcpServer.AcceptLoop()

	err := InitFromConfig(config)
	if err != nil {
		fmt.Println("InitFromConfig error", err)
		os.Exit(1)
	}
	SetSrvHandler("Server1", &TCPServerHandlers{UnpackHandler: xmandemo.UnpackTCP, PackageHandler: TCPLogicHandler})
	SetSrvHandler("Server2", &UDPServerHandlers{UnpackHandler: xmandemo.UnpackUDP, PackageHandler: UDPLogicHandler})
	//	SetConnHandler("Conn1", &TCPConnHandlers{PackHandler: xmandemo.PackConn, UnpackHandler:xmandemo.UnpackTCPConn})
	SetReloadHandler(ReloadConfigure)

	RunServer()
}

func main() {
	fmt.Println("XMAN Test")

	// argv: name default_val introduction
	configFile := flag.String("conf", "./config/srv.ini", "server config file.")
	flag.Parse()

	fmt.Println(*configFile)
	config := InitConfig(*configFile)
	section, err := config.Section("MysqlInfo")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(section.ValueOf("host"))

	test := &xmandemo.BaseHeader{
		Ver: proto.Uint32(111),
		Bus: proto.Uint32(222),
		Seq: proto.Uint32(1),
		Cmd: proto.Uint32(uint32(xmandemo.XCMD_CMD_DEMO)),
	}
	data, err := proto.Marshal(test)
	fmt.Println(data)

	//	utils.Daemonize(0, 1)

	SetConsoleShow(true)
	//	SetRollingDaily("/Users/reezhou/Desktop/xman/src/logs", "test.log")
	SetRollingFile("/Users/reezhou/Desktop/xman/src/logs", "rolling.log", 10, 50, MB)
	SetLogLevel(LOG_DEBUG)
	//	Log(LOG_ERROR, "uin error")

	// redis test
	//	redisController = xmandb.NewRedisController()
	//	defer redisController.Close()
	//	err = redisController.InitRedis("", ":6379", 0)
	//	if err != nil {
	//		fmt.Println("redis init error.",err)
	//	}
	//	v, _ := redis.String(redisController.Get("ree"), err)
	//	fmt.Println("get from redis: ", v)
	//
	//	// mysql test
	//	mysqlController = xmandb.NewMysqlController()
	//	err = mysqlController.InitMysql(`{"user": "root", "password": "111", "address": ":3306", "dbname": "xman"}`)
	//	defer mysqlController.Close()
	//	if err != nil {
	//		fmt.Println(err)
	//	}
	//	row, _ := mysqlController.FetchRows("SELECT * FROM user")
	//	fmt.Println(row)

	goServer(config)
}
