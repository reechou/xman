package controller

import (
	"strconv"

	"db"
	"fmt"
	"strings"
)

type MysqlLogic struct {
	mysqlController *xmandb.MysqlController
}

func NewMysqlLogic(dbinfo string) (*MysqlLogic, error) {
	mysqlLogic := &MysqlLogic{mysqlController: xmandb.NewMysqlController()}
	err := mysqlLogic.mysqlController.InitMysql(dbinfo)
	if err != nil {
		return nil, err
	}

	return mysqlLogic, nil
}

func (ml *MysqlLogic) GetIPList(start int, num int, srvList *ServerList) error {
	rows, err := ml.mysqlController.FetchRows("SELECT ip FROM machine_ip limit " + strconv.Itoa(start) + "," + strconv.Itoa(num))
	if err != nil {
		return err
	}
	for _, v := range *rows {
		srv := &Server{
			IP:     v["ip"],
			Status: SRV_STATUS_OK,
		}
		srvList.SrvList = append(srvList.SrvList, srv)
	}

	return nil
}

func (ml *MysqlLogic) GetIPMaxID(srvList *ServerList) error {
	row, err := ml.mysqlController.FetchRow("SELECT count(id) as max_id FROM machine_ip")
	if err != nil {
		return err
	}
	srvList.SrvAllNum, _ = strconv.Atoi((*row)["max_id"])
	return nil
}

func (ml *MysqlLogic) SearchIP(likeIP string, srvList *ServerList) error {
	rows, err := ml.mysqlController.FetchRows("SELECT ip FROM machine_ip where ip like '%" + likeIP + "%'")
	if err != nil {
		return err
	}
	num := 0
	for _, v := range *rows {
		srv := &Server{
			IP:     v["ip"],
			Status: SRV_STATUS_OK,
		}
		srvList.SrvList = append(srvList.SrvList, srv)
		num++
		if num >= 10 {
			break
		}
	}
	srvList.SrvAllNum = num

	return nil
}

func (ml *MysqlLogic) AddIP(ip string) error {
	_, err := ml.mysqlController.Insert("insert into machine_ip(ip) values('" + ip + "')")
	if err != nil {
		return err
	}

	return nil
}

func (ml *MysqlLogic) DelIP(ip string) error {
	_, err := ml.mysqlController.Exec("delete from machine_ip where ip='" + ip + "'")
	if err != nil {
		return err
	}

	return nil
}

func (ml *MysqlLogic) SelectTestModule(start int, num int, testList *TestList) error {
	rows, err := ml.mysqlController.FetchRows("SELECT id,ip,src_dir,dst_dir,module_name,principal,version,update_time FROM code_test limit " + strconv.Itoa(start) + "," + strconv.Itoa(num))
	if err != nil {
		return err
	}
	for _, v := range *rows {
		testModule := &TestModule{
			ID:         v["id"],
			IP:         v["ip"],
			SrcDir:     v["src_dir"],
			DstDir:     v["dst_dir"],
			ModuleName: v["module_name"],
			Principal:  v["principal"],
			Version:    v["version"],
			UpdateTime: v["update_time"],
		}
		testList.TestModuleList = append(testList.TestModuleList, testModule)
	}

	return nil
}

func (ml *MysqlLogic) SelectTestModuleCommand(id string, testCommand *TestModuleCommand) error {
	row, err := ml.mysqlController.FetchRow("SELECT compile_commad,restart_commad FROM code_test where id=" + id)
	if err != nil {
		return err
	}
	testCommand.CompileCommand = (*row)["compile_commad"]
	testCommand.RestartCommand = (*row)["restart_commad"]

	return nil
}

func (ml *MysqlLogic) UpdateCompileCommand(cmd string, id string) error {
	_, err := ml.mysqlController.Exec("update code_test set compile_commad='" + cmd + "' where id=" + id)
	if err != nil {
		return err
	}

	return nil
}

func (ml *MysqlLogic) UpdateRestartCommand(cmd string, id string) error {
	_, err := ml.mysqlController.Exec("update code_test set restart_commad='" + cmd + "' where id=" + id)
	if err != nil {
		return err
	}

	return nil
}

func (ml *MysqlLogic) SelectMaxIDTestModule(testList *TestList) error {
	row, err := ml.mysqlController.FetchRow("SELECT count(id) as max_id FROM code_test")
	if err != nil {
		return err
	}
	testList.TestModuleNum, _ = strconv.Atoi((*row)["max_id"])
	return nil
}

func (ml *MysqlLogic) AddTestModule(testModule *TestModule) error {
	sql := "insert into code_test(ip,src_dir,dst_dir,module_name,principal,version) values('" + testModule.IP + "','" + testModule.SrcDir + "','" + testModule.DstDir + "','" + testModule.ModuleName + "','" + testModule.Principal + "'," + testModule.Version + ")"
	_, err := ml.mysqlController.Insert(sql)
	if err != nil {
		return err
	}

	return nil
}

// about paidang
func (ml *MysqlLogic) AddPaidang(paidang *Paidang) error {
	sql := "insert into paidang(name,phone,location,user) values('" + paidang.Name + "','" + paidang.Phone + "','" + paidang.Location + "','" + paidang.User + "')"
	_, err := ml.mysqlController.Insert(sql)
	if err != nil {
		return err
	}

	return nil
}
func (ml *MysqlLogic) DelPaidang(id string) error {
	_, err := ml.mysqlController.Exec("delete from paidang where id=" + id)
	if err != nil {
		return err
	}

	return nil
}
func (ml *MysqlLogic) GetPaidangList(start int, num int, paidangList *PaidangList) error {
	rows, err := ml.mysqlController.FetchRows("SELECT id,name,phone,location,user,update_time FROM paidang limit " + strconv.Itoa(start) + "," + strconv.Itoa(num))
	if err != nil {
		return err
	}
	for _, v := range *rows {
		paidang := &Paidang{
			ID:       v["id"],
			Name:     v["name"],
			Phone:    v["phone"],
			Location: v["location"],
			User:     v["user"],
			Time:     v["update_time"],
		}
		paidangList.PaidangInfoList = append(paidangList.PaidangInfoList, paidang)
	}

	return nil
}
func (ml *MysqlLogic) GetPaidangCount(paidangList *PaidangList) error {
	row, err := ml.mysqlController.FetchRow("SELECT count(id) as max_id FROM paidang")
	if err != nil {
		return err
	}
	paidangList.PaidangNum, _ = strconv.Atoi((*row)["max_id"])
	return nil
}
func (ml *MysqlLogic) SearchPaidang(location, startTime, endTime string, paidangList *PaidangList) error {
	sql := ""
	if endTime == "" {
		sql = "SELECT id,name,phone,location,user,update_time FROM paidang where location like '%" + location + "%' and update_time>='" + startTime + "'"
	} else {
		sql = "SELECT id,name,phone,location,user,update_time FROM paidang where location like '%" + location + "%' and update_time>='" + startTime + "' and update_time<='" + endTime + "'"
	}
	sql += " GROUP BY phone"
	fmt.Println(sql)
	rows, err := ml.mysqlController.FetchRows(sql)
	if err != nil {
		return err
	}
	num := 0
	for _, v := range *rows {
		paidang := &Paidang{
			ID:       v["id"],
			Name:     v["name"],
			Phone:    v["phone"],
			Location: v["location"],
			User:     v["user"],
			Time:     v["update_time"],
		}
		paidangList.PaidangInfoList = append(paidangList.PaidangInfoList, paidang)
		num++
	}
	paidangList.PaidangNum += num

	return nil
}

func (ml *MysqlLogic) SearchPaidangNot(location, startTime, endTime string, paidangList *PaidangList) error {
	sql := "SELECT id,name,phone,location,user,update_time FROM paidang where not location in ("
	locaSlice := strings.Split(location, " ")
	len := len(locaSlice)
	for idx, v := range locaSlice {
		if idx != len-1 {
			sql += "'" + v + "',"
		} else {
			sql += "'" + v + "')"
		}
	}
	if endTime == "" {
		sql += " and update_time>='" + startTime + "'"
	} else {
		sql += " and update_time>='" + startTime + "' and update_time<='" + endTime + "'"
	}
	sql += " GROUP BY phone"
	fmt.Println(sql)
	rows, err := ml.mysqlController.FetchRows(sql)
	if err != nil {
		return err
	}
	num := 0
	for _, v := range *rows {
		paidang := &Paidang{
			ID:       v["id"],
			Name:     v["name"],
			Phone:    v["phone"],
			Location: v["location"],
			User:     v["user"],
			Time:     v["update_time"],
		}
		paidangList.PaidangInfoList = append(paidangList.PaidangInfoList, paidang)
		num++
	}
	paidangList.PaidangNum += num

	return nil
}

// about modules
func (ml *MysqlLogic) QueryHauntModuleList(start int, num int, moduleList *HauntModuleList) error {
	rows, err := ml.mysqlController.FetchRows("SELECT id,module_name,module_desc,module_user,module_backup_user,update_time FROM modules limit " + strconv.Itoa(start) + "," + strconv.Itoa(num))
	if err != nil {
		return err
	}
	for _, v := range *rows {
		module := &HauntModule{
			ID:               v["id"],
			ModuleName:       v["module_name"],
			ModuleDesc:       v["module_desc"],
			ModuleUser:       v["module_user"],
			ModuleBackupUser: v["module_backup_user"],
			UpdateTime:       v["update_time"],
		}
		moduleList.ModuleList = append(moduleList.ModuleList, module)
	}
	return nil
}
func (ml *MysqlLogic) QueryHauntModuleCount(moduleList *HauntModuleList) error {
	row, err := ml.mysqlController.FetchRow("SELECT count(id) as module_num FROM modules")
	if err != nil {
		return err
	}
	moduleList.ModuleNum, _ = strconv.Atoi((*row)["module_num"])
	return nil
}
func (ml *MysqlLogic) AddHauntModule(hauntModule *HauntModule) error {
	sql := "insert into modules(module_name,module_desc,module_user,module_backup_user) values('" + hauntModule.ModuleName + "','" + hauntModule.ModuleDesc + "','" + hauntModule.ModuleUser + "','" + hauntModule.ModuleBackupUser + "')"
	_, err := ml.mysqlController.Insert(sql)
	if err != nil {
		return err
	}

	return nil
}
func (ml *MysqlLogic) DelHauntModule(id string) error {
	_, err := ml.mysqlController.Exec("delete from modules where id=" + id)
	if err != nil {
		return err
	}

	return nil
}
func (ml *MysqlLogic) UpdateHauntModule(hauntModule *HauntModule) error {
	sql := "update modules set module_desc='" + hauntModule.ModuleDesc + "', module_user='" + hauntModule.ModuleUser + "', module_backup_user='" + hauntModule.ModuleBackupUser + "' where module_name='" + hauntModule.ModuleName + "'"
	_, err := ml.mysqlController.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}
