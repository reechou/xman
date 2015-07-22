
package main

import (
	"fmt"
	"net"
	"os"
//	"time"

	. "demo"
	"time"
)

func udpClient() {
	udpAddr, err := net.ResolveUDPAddr("udp", ":8000")
	checkError(err)
	conn, err := net.DialUDP("udp", nil, udpAddr)
	checkError(err)
	pkg := &ProtoPkg{
		Header: new(ProtoHeader),
		Body: []byte("I am a bad boy"),
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

func tcpClient() {
	server := ":7000"
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

	pkg := &ProtoPkg{
		Header: new(ProtoHeader),
		Body: []byte("I am a good boy"),
	}
	pkg.Header.Version = Version
	pkg.Header.Cmd = 111
	pkgData := Pack(pkg)
	fmt.Println(pkg, pkgData)
	for i := 0; i < 1; i++ {
		conn.Write(pkgData)
	}

	time.Sleep(2 * 1e9)
}

func main() {
	fmt.Println("--tcpclient--")
	for i := 0; i < 1; i++ {
		tcpClient()
	}
	fmt.Println("--udpclient--")
	udpClient()
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
		os.Exit(1)
	}
}
