// copyright xman
// author    reezhou
// email     reechou@gmail.com
// file      xman_const_define.go

package xman

const sendChanBufLen = 1024
const recvChanBufLen = 1024
const MaxReqChanLen  = 10000
const RecvReqBufLen  = 1024

const tcpReadBufLen  = 64 * 1024
const tcpWriteBufLen = 64 * 1024

const maxSrvNum      = 16
const maxRemoteSrvIP = 16

const sessionCheck   = 10
const sessionTimeout = 120
const maxSessionSeq  = 10000
const maxUINT32      = 4200000000

const tcpConnCheckReconnect = 10

type RET_CODE int32
const (
	RET_OK = iota
	RET_TIMEOUT
	RET_NETWORK_ERR
)

const (
	TCP = 1
	UDP = 2
)
