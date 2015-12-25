package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	. "demo"

	//	haunt "git.qima-inc.com/zhoulindong/haunt/name_service_proxy/namesrv_proxy"
	haunt "youzan/haunt/name_service_proxy/namesrv_proxy"
)

func udpClient() {
	udpAddr, err := net.ResolveUDPAddr("udp", ":8000")
	checkError(err)
	conn, err := net.DialUDP("udp", nil, udpAddr)
	checkError(err)
	pkg := &ProtoPkg{
		Header: new(ProtoHeader),
		Body:   []byte("I am a bad boy"),
	}
	pkg.Header.Version = Version
	pkg.Header.Cmd = 111
	pkgData := Pack(pkg)
	_, err = conn.Write(pkgData)
	checkError(err)
	var buf [512]byte
	n, err := conn.Read(buf[0:])
	checkError(err)
	fmt.Println(string(buf[0:n]))
}

type SeqInfo struct {
	t      time.Time
	server string
}

var (
	seqMap map[uint32]*SeqInfo
	seq    uint32
)

func checkTimeout() {
	for k, v := range seqMap {
		waitTime := time.Since(v.t)
		millTime := float64(waitTime.Nanoseconds()) / float64(1000000)
		if millTime >= 200.0 {
			//			fmt.Println(seq, "timeout.")
			haunt.SetRouteResult("7900_xman", v.server, 1, 200)
			delete(seqMap, k)
		}
	}
}
func handleReader(conn net.Conn, server string) {
	tmpbuf := make([]byte, 0)
	for {
		buf := make([]byte, 1024)
		reqLen, err := conn.Read(buf)
		if err != nil {
			fmt.Println(err)
			continue
		}
		sMap := make(map[uint32]interface{})
		tmpbuf = UnpackTCPConn(append(tmpbuf, buf[:reqLen]...), &sMap)
		for k, _ := range sMap {
			v, ok := seqMap[k]
			if ok {
				useTime := time.Since(v.t)
				//				fmt.Println(ip, "use time:", useTime, float64(useTime.Nanoseconds())/float64(1000000))
				haunt.SetRouteResult("7900_xman", server, 0, float64(useTime.Nanoseconds())/float64(1000000))
				delete(seqMap, k)
			}
		}
	}
}
func tcpWorker(conn *net.TCPConn, server string) {
	go handleReader(conn, server)
	pkg := &ProtoPkg{
		Header: new(ProtoHeader),
		Body:   []byte("I am a good boy"),
	}
	pkg.Header.Version = Version
	pkg.Header.Cmd = 111
	//	fmt.Println(pkg, pkgData)
	for {
		for i := 0; i < 10; i++ {
			seq = atomic.AddUint32(&seq, 1)
			pkg.Header.Seq = seq
			seqInfo := &SeqInfo{
				t:      time.Now(),
				server: server,
			}
			seqMap[seq] = seqInfo
			pkgData := Pack(pkg)
			conn.Write(pkgData)
		}
		checkTimeout()
		time.Sleep(100 * time.Millisecond)
	}
}

func tcpClient() {
	// get srv list from haunt
	hauntRet, err := haunt.Get([]string{"7900_xman"})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	var srvList []*haunt.JsonValue
	json.Unmarshal(hauntRet, &srvList)

	var wg sync.WaitGroup
	for _, v := range srvList[0].SrvList {
		//		if v.IP != "192.168.66.205" {
		//			continue
		//		}
		server := fmt.Sprintf("%s:%d", v.IP, v.Port)
		fmt.Println("server:", server)
		tcpAddr, err := net.ResolveTCPAddr("tcp", server)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
			os.Exit(1)
		}

		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
			os.Exit(1)
		}

		defer conn.Close()
		fmt.Println("connect success")

		wg.Add(1)
		go tcpWorker(conn, server)
	}
	wg.Wait()
}

func main() {
	// init haunt
	haunt.InitHaunt("./youzan/haunt/name_service_proxy/namesrv_proxy.ini")

	seqMap = make(map[uint32]*SeqInfo)
	seq = 0
	// test haunt
	tcpClient()
	//	fmt.Println("--udpclient--")
	//	udpClient()
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
		os.Exit(1)
	}
}
