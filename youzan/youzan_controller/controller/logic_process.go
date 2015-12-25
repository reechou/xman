package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	. "youzan/haunt/namesrv_utils"
)

const (
	SRV_STATUS_NONE = iota
	SRV_STATUS_OK
	SRV_STATUS_NO_OK
)

type LogicHandler struct {
	httpSrv        *HttpSrv
	etcdExecClient *EtcdClient
}

func NewLogicHandler(etcdPeers string) *LogicHandler {
	logic := &LogicHandler{
		//		etcdExecClient: NewEtcdClient("192.168.66.205:2379,192.168.66.237:2379"),
		etcdExecClient: NewEtcdClient(etcdPeers),
		httpSrv:        &HttpSrv{HttpAddr: "", HttpPort: 2323, Routers: make(map[string]http.HandlerFunc)},
	}
	logic.httpSrv.Route("/getsrvlist", GetSrvList)
	logic.httpSrv.Route("/searchip", SearchIP)
	logic.httpSrv.Route("/addip", AddMachine)
	logic.httpSrv.Route("/delip", DelMachine)

	logic.httpSrv.Route("/gettestmodule", GetTestModuleList)
	logic.httpSrv.Route("/addtestmodule", AddTestModule)
	logic.httpSrv.Route("/execremotecommand", ExecRemoteCommand)
	logic.httpSrv.Route("/getcommands", SelectCommands)
	logic.httpSrv.Route("/updatecompilecommand", UpdateCompileCommand)
	logic.httpSrv.Route("/updaterestartcommand", UpdateRestartCommand)
	logic.httpSrv.Route("/synccode", SyncTestCode)

	logic.httpSrv.Route("/addpaidang", AddPaidang)
	logic.httpSrv.Route("/getpaidanglist", GetPaidangList)
	logic.httpSrv.Route("/searchpaidang", SearchPaidang)
	logic.httpSrv.Route("/delpaidang", DelPaidang)
	logic.httpSrv.Route("/searchnotpaidang", SearchNotPaidang)

	logic.httpSrv.Route("/gethauntmodules", GetHauntModules)
	logic.httpSrv.Route("/gethauntsrvlist", GetHauntSrvList)
	logic.httpSrv.Route("/addhauntmodule", AddHauntModule)
	logic.httpSrv.Route("/delhauntmodule", DelHauntModule)
	logic.httpSrv.Route("/updatehauntmodule", UpdateHauntModule)
	logic.httpSrv.Route("/hauntaddsrvip", AddSrvIP)
	logic.httpSrv.Route("/hauntdelsrvip", DelSrvIP)

	logic.httpSrv.Route("/hauntroot", HauntHTTP.httpWrap(HauntHTTP.GetRoot))
	logic.httpSrv.Route("/hauntmodules", HauntHTTP.httpWrap(HauntHTTP.GetModules))

	return logic
}

func (lh *LogicHandler) Run() {
	lh.httpSrv.Run()
}

func (lh *LogicHandler) HttpGet(url string) error {
	fmt.Println("HttpGet:", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func (lh *LogicHandler) GetRoot() ([]*RootList, error) {
	resp, err := lh.etcdExecClient.Get("/")
	if err != nil {
		fmt.Println("GetRoot error:", err.Error())
		return nil, err
	}
	rootList := make([]*RootList, 0)
	if len(resp.Node.Nodes) == 0 {
		fmt.Println("GetRoot len==0")
		return nil, nil
	}
	for _, node := range resp.Node.Nodes {
		rootKey := &RootList{}
		rootKey.IfDir = node.Dir
		rootKey.Key = node.Key
		rootList = append(rootList, rootKey)
	}
	return rootList, nil
}

func (lh *LogicHandler) GetModules(ns string) ([]*RootList, error) {
	resp, err := lh.etcdExecClient.Get("/"+ns)
	if err != nil {
		fmt.Println("GetModules error:", err.Error())
		return nil, err
	}
	rootList := make([]*RootList, 0)
	if len(resp.Node.Nodes) == 0 {
		fmt.Println("GetModuels len==0")
		return nil, nil
	}
	for _, node := range resp.Node.Nodes {
		rootKey := &RootList{}
		rootKey.IfDir = node.Dir
		rootKey.Key = node.Key
		rootList = append(rootList, rootKey)
	}
	return rootList, nil
}

func (lh *LogicHandler) GetModuleSrvList(key string) ([]*HauntSrvInfo, error) {
	etcdKey := "/" + key
	resp, err := lh.etcdExecClient.Get(etcdKey)
	if err != nil {
		fmt.Println("etcdExecClient.Get error:", err.Error(), "key:", key)
		return nil, nil
	}
	hauntSrvList := make([]*HauntSrvInfo, 0)
	if len(resp.Node.Nodes) == 0 {
		fmt.Println("len(resp.Node.Nodes) == 0", "key:", key)
		return nil, nil
	}
	for _, node := range resp.Node.Nodes {
		var moduleValue HauntSrvInfo
		json.Unmarshal([]byte(node.Value), &moduleValue)
		hauntSrvList = append(hauntSrvList, &moduleValue)
	}
	return hauntSrvList, nil
}

func (lh *LogicHandler) AddSrvIP(hauntSrvInfo *HauntSrvIPInfo) error {
	url := "http://" + hauntSrvInfo.IP + ":8082/registerp?IP=" + hauntSrvInfo.IP + "&Port=" + hauntSrvInfo.Port + "&SrvName=" + hauntSrvInfo.SrvName + "&Weight=" + hauntSrvInfo.Weight
	err := lh.HttpGet(url)
	if err != nil {
		return err
	}
	return nil
}

func (lh *LogicHandler) DelSrvIP(hauntSrvInfo *HauntSrvIPInfo) error {
	url := "http://" + hauntSrvInfo.IP + ":8082/unregisterp?IP=" + hauntSrvInfo.IP + "&Port=" + hauntSrvInfo.Port + "&SrvName=" + hauntSrvInfo.SrvName
	err := lh.HttpGet(url)
	if err != nil {
		return err
	}
	return nil
}
