package controller

import (
	"fmt"
	"os"

	"github.com/reechou/xman/config"
)

var (
	ConfigPath string
	EtcdPeers  string
	MysqlHost  string
	MysqlUser  string
	MysqlPass  string
	MysqlDB    string

	config *xmanconfig.Configuration
)

func InitConfig() {
	if ConfigPath == "" {
		ConfigPath = "./youzan_controller.ini"
	}
	conf, err := xmanconfig.Read(ConfigPath)
	if err != nil {
		fmt.Println("Config read error:", err.Error(), ConfigPath)
		os.Exit(1)
	}
	config = conf

	mysqlSection, err := config.Section("MysqlInfo")
	if err != nil {
		fmt.Println("MysqlInfo read error:", err.Error(), ConfigPath)
		os.Exit(1)
	}
	MysqlHost = mysqlSection.ValueOf("MysqlHost")
	MysqlUser = mysqlSection.ValueOf("MysqlUser")
	MysqlPass = mysqlSection.ValueOf("MysqlPass")
	MysqlDB = mysqlSection.ValueOf("MysqlDB")

	etcdSection, err := config.Section("EtcdInfo")
	if err != nil {
		fmt.Println("EtcdInfo read error:", err.Error(), ConfigPath)
		os.Exit(1)
	}
	EtcdPeers = etcdSection.ValueOf("EtcdPeers")

	dbinfo := `{"user": "` + MysqlUser + `", "password": "` + MysqlPass + `", "address": "` + MysqlHost + `", "dbname": "` + MysqlDB + `"}`
	//	mysqlLogic, err := NewMysqlLogic(`{"user": "test_koudaitong", "password": "nPMj9WWpZr4zNmjz", "address": "192.168.66.202:3306", "dbname": "test_youzan_ree"}`)
	mysqlLogic, err := NewMysqlLogic(dbinfo)
	if err != nil {
		fmt.Println(err)
	}
	mysqlHander = mysqlLogic
	Logic = NewLogicHandler(EtcdPeers)
	HauntHTTP = &HauntHttpLogic{}
}
