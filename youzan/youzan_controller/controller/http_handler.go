package controller

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	mysqlHander *MysqlLogic
	Logic       *LogicHandler
	HauntHTTP   *HauntHttpLogic
)

type Server struct {
	IP     string
	Status int
}
type ServerList struct {
	SrvAllNum int
	SrvList   []*Server
}

type TestModule struct {
	ID         string
	IP         string
	SrcDir     string
	DstDir     string
	ModuleName string
	Principal  string
	Version    string
	UpdateTime string
}
type TestList struct {
	TestModuleNum  int
	TestModuleList []*TestModule
}

type TestModuleCommand struct {
	CompileCommand string
	RestartCommand string
}

type Paidang struct {
	ID       string
	Name     string
	Phone    string
	Location string
	User     string
	Time     string
}
type PaidangList struct {
	PaidangNum      int
	PaidangInfoList []*Paidang
}

type HauntModule struct {
	ID               string
	ModuleName       string
	ModuleDesc       string
	ModuleUser       string
	ModuleBackupUser string
	UpdateTime       string
}
type HauntModuleList struct {
	ModuleNum  int
	ModuleList []*HauntModule
}
type HauntSrvInfo struct {
	SrvName string
	IP      string
	Port    int
	Status  int
	Weight  int
}

type HauntSrvIPInfo struct {
	IP      string
	Port    string
	SrvName string
	Weight  string
}

func CheckIPAlive(srvList *ServerList) {
	for _, v := range srvList.SrvList {
		srvStr := fmt.Sprintf("%s:22", v.IP)
		_, err := net.DialTimeout("tcp", srvStr, 100*time.Millisecond)
		if err != nil {
			v.Status = SRV_STATUS_NO_OK
		}
	}
}

func InitHttpHandler() {
	mysqlLogic, err := NewMysqlLogic(`{"user": "test_koudaitong", "password": "nPMj9WWpZr4zNmjz", "address": "192.168.66.202:3306", "dbname": "test_youzan_ree"}`)
	if err != nil {
		fmt.Println(err)
	}
	mysqlHander = mysqlLogic
	Logic = NewLogicHandler(EtcdPeers)
}

func GetSrvList(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	start, _ := strconv.Atoi(r.Form["start"][0])
	num, _ := strconv.Atoi(r.Form["num"][0])
	srvList := &ServerList{SrvList: make([]*Server, 0)}
	mysqlHander.GetIPMaxID(srvList)
	mysqlHander.GetIPList(start, num, srvList)
	CheckIPAlive(srvList)
	resultsBytes, err := json.Marshal(srvList)
	if err != nil {
		return
	}
	callback := r.FormValue("callback")
	resultsBytes = []byte(fmt.Sprintf("%s(%s)", callback, resultsBytes))
	rw.Write(resultsBytes)
}

func SearchIP(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	srvList := &ServerList{SrvList: make([]*Server, 0)}
	mysqlHander.SearchIP(r.Form["searchip"][0], srvList)
	CheckIPAlive(srvList)
	resultsBytes, err := json.Marshal(srvList)
	if err != nil {
		return
	}
	callback := r.FormValue("callback")
	resultsBytes = []byte(fmt.Sprintf("%s(%s)", callback, resultsBytes))
	rw.Write(resultsBytes)
}

func AddMachine(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	err := mysqlHander.AddIP(r.Form["ip"][0])
	retMsg := "{\"result\": \"Add IP OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"Add IP Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}

func DelMachine(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	err := mysqlHander.DelIP(r.Form["ip"][0])
	retMsg := "{\"result\": \"Delete IP OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"Delete IP Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}

func GetTestModuleList(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	start, _ := strconv.Atoi(r.Form["start"][0])
	num, _ := strconv.Atoi(r.Form["num"][0])
	testList := &TestList{TestModuleList: make([]*TestModule, 0)}
	mysqlHander.SelectMaxIDTestModule(testList)
	mysqlHander.SelectTestModule(start, num, testList)
	resultsBytes, err := json.Marshal(testList)
	if err != nil {
		return
	}
	callback := r.FormValue("callback")
	resultsBytes = []byte(fmt.Sprintf("%s(%s)", callback, resultsBytes))
	rw.Write(resultsBytes)
}

func AddTestModule(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	testModule := &TestModule{
		IP:         r.Form["ip"][0],
		SrcDir:     r.Form["srcdir"][0],
		DstDir:     r.Form["dstdir"][0],
		ModuleName: r.Form["modulename"][0],
		Principal:  r.Form["user"][0],
		Version:    strconv.Itoa(1),
	}
	err := mysqlHander.AddTestModule(testModule)
	retMsg := "{\"result\": \"Add TestModule OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"Add TestModule Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}

func SyncTestCode(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	srcDir := r.Form["srcdir"][0]
	dstDir := r.Form["dstdir"][0]
	ccc := `ansible test -m copy -a "src=` + srcDir + ` dest=` + dstDir + `" -i /Users/reezhou/Desktop/youzan/ssh/REE -u zhoulindong`
	//	fmt.Println(ccc)
	cmd := exec.Command("/bin/sh", "-c", ccc)
	_, err := cmd.Output()
	retMsg := ""
	if err != nil {
		retMsg = "{\"result\": \"SyncTestCode error, " + err.Error() + ".\"}"
	} else {
		retMsg = "{\"result\": \"SyncTestCode OK.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}

func ExecRemoteCommand(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	command := r.Form["command"][0]
	ccc := `ansible test -m shell -a "` + command + `" -i /Users/reezhou/Desktop/youzan/ssh/REE -u zhoulindong`
	cmd := exec.Command("/bin/sh", "-c", ccc)
	buf, err := cmd.Output()
	retMsg := ""
	if err != nil {
		retMsg = "{\"result\": \"ExecRemoteCommand error, " + err.Error() + ".\"}"
	} else {
		retMsg = "{\"result\": \"ExecRemoteCommand OK.\", \"msg\": \"" + base64.StdEncoding.EncodeToString(buf) + "\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}

func SelectCommands(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	testCommand := &TestModuleCommand{}
	mysqlHander.SelectTestModuleCommand(r.Form["id"][0], testCommand)
	resultsBytes, err := json.Marshal(testCommand)
	if err != nil {
		return
	}
	callback := r.FormValue("callback")
	resultsBytes = []byte(fmt.Sprintf("%s(%s)", callback, resultsBytes))
	rw.Write(resultsBytes)
}

func UpdateCompileCommand(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	err := mysqlHander.UpdateCompileCommand(r.Form["cmd"][0], r.Form["id"][0])
	retMsg := "{\"result\": \"UpdateCompileCommand OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"UpdateCompileCommand Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}

func UpdateRestartCommand(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	err := mysqlHander.UpdateRestartCommand(r.Form["cmd"][0], r.Form["id"][0])
	retMsg := "{\"result\": \"UpdateRestartCommand OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"UpdateRestartCommand Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}

// about paidang
func GetPaidangList(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	start, _ := strconv.Atoi(r.Form["start"][0])
	num, _ := strconv.Atoi(r.Form["num"][0])
	paidangList := &PaidangList{PaidangInfoList: make([]*Paidang, 0)}
	mysqlHander.GetPaidangCount(paidangList)
	mysqlHander.GetPaidangList(start, num, paidangList)
	resultsBytes, err := json.Marshal(paidangList)
	if err != nil {
		return
	}
	callback := r.FormValue("callback")
	resultsBytes = []byte(fmt.Sprintf("%s(%s)", callback, resultsBytes))
	rw.Write(resultsBytes)
}
func SearchPaidang(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	paidangList := &PaidangList{PaidangInfoList: make([]*Paidang, 0)}
	locaSlice := strings.Split(r.Form["location"][0], " ")
	for _, v := range locaSlice {
		mysqlHander.SearchPaidang(v, r.Form["starttime"][0], r.Form["endtime"][0], paidangList)
	}
	resultsBytes, err := json.Marshal(paidangList)
	if err != nil {
		return
	}
	callback := r.FormValue("callback")
	resultsBytes = []byte(fmt.Sprintf("%s(%s)", callback, resultsBytes))
	rw.Write(resultsBytes)
}
func SearchNotPaidang(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	paidangList := &PaidangList{PaidangInfoList: make([]*Paidang, 0)}
	mysqlHander.SearchPaidangNot(r.Form["location"][0], r.Form["starttime"][0], r.Form["endtime"][0], paidangList)
	resultsBytes, err := json.Marshal(paidangList)
	if err != nil {
		return
	}
	callback := r.FormValue("callback")
	resultsBytes = []byte(fmt.Sprintf("%s(%s)", callback, resultsBytes))
	rw.Write(resultsBytes)
}
func AddPaidang(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	paidang := &Paidang{
		Name:     r.Form["name"][0],
		Phone:    r.Form["phone"][0],
		Location: r.Form["location"][0],
		User:     r.Form["user"][0],
	}
	err := mysqlHander.AddPaidang(paidang)
	retMsg := "{\"result\": \"AddPaidang OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"AddPaidang Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}
func DelPaidang(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	err := mysqlHander.DelPaidang(r.Form["id"][0])
	retMsg := "{\"result\": \"DelPaidang OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"DelPaidang Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}

// about haunt modules
func GetHauntModules(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	start, _ := strconv.Atoi(r.Form["start"][0])
	num, _ := strconv.Atoi(r.Form["num"][0])
	moduleList := &HauntModuleList{ModuleList: make([]*HauntModule, 0)}
	mysqlHander.QueryHauntModuleCount(moduleList)
	mysqlHander.QueryHauntModuleList(start, num, moduleList)
	resultsBytes, err := json.Marshal(moduleList)
	if err != nil {
		return
	}
	callback := r.FormValue("callback")
	resultsBytes = []byte(fmt.Sprintf("%s(%s)", callback, resultsBytes))
	rw.Write(resultsBytes)
}
func GetHauntSrvList(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	srvList, err := Logic.GetModuleSrvList(r.Form["key"][0])
	if err != nil {
		return
	}
	resultsBytes, err := json.Marshal(srvList)
	if err != nil {
		return
	}
	callback := r.FormValue("callback")
	resultsBytes = []byte(fmt.Sprintf("%s(%s)", callback, resultsBytes))
	rw.Write(resultsBytes)
}
func AddHauntModule(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	hauntModule := &HauntModule{
		ModuleName:       r.Form["ModuleName"][0],
		ModuleDesc:       r.Form["ModuleDesc"][0],
		ModuleUser:       r.Form["ModuleUser"][0],
		ModuleBackupUser: r.Form["ModuleBackupUser"][0],
	}
	err := mysqlHander.AddHauntModule(hauntModule)
	retMsg := "{\"result\": \"AddHauntModule OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"AddHauntModule Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}
func DelHauntModule(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	err := mysqlHander.DelHauntModule(r.Form["id"][0])
	retMsg := "{\"result\": \"DelHauntModule OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"DelHauntModule Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}
func UpdateHauntModule(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	hauntModule := &HauntModule{
		ModuleName:       r.Form["ModuleName"][0],
		ModuleDesc:       r.Form["ModuleDesc"][0],
		ModuleUser:       r.Form["ModuleUser"][0],
		ModuleBackupUser: r.Form["ModuleBackupUser"][0],
	}
	err := mysqlHander.UpdateHauntModule(hauntModule)
	retMsg := "{\"result\": \"UpdateHauntModule OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"UpdateHauntModule Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}
func AddSrvIP(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	hauntSrvInfo := &HauntSrvIPInfo{
		IP:      r.Form["IP"][0],
		Port:    r.Form["Port"][0],
		SrvName: r.Form["SrvName"][0],
		Weight:  r.Form["Weight"][0],
	}
	err := Logic.AddSrvIP(hauntSrvInfo)
	retMsg := "{\"result\": \"AddSrvIP OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"AddSrvIP Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}
func DelSrvIP(rw http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	hauntSrvInfo := &HauntSrvIPInfo{
		IP:      r.Form["IP"][0],
		Port:    r.Form["Port"][0],
		SrvName: r.Form["SrvName"][0],
	}
	err := Logic.DelSrvIP(hauntSrvInfo)
	retMsg := "{\"result\": \"DelSrvIP OK\"}"
	if err != nil {
		retMsg = "{\"result\": \"DelSrvIP Error.\"}"
	}
	callback := r.FormValue("callback")
	resultsBytes := []byte(fmt.Sprintf("%s(%s)", callback, retMsg))
	rw.Write(resultsBytes)
}

