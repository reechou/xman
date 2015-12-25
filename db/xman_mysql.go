// copyright xman
// author    reezhou
// email     reechou@gmail.com
// file      xman_mysql.go

package xmandb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"

	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

var (
	DefaultOpenConns = 200
	DefaultIdleConns = 100
	DefaultUser      = "root"
)

var (
	ErrMysqlNoHost   = errors.New("Mysql has no host.")
	ErrMysqlNoDBName = errors.New("Mysql has no dbname.")
	ErrMysqlNotInit  = errors.New("Mysql not init.")
)

const (
	SQL_INSERT = "INSERT INTO %s (%s) VALUES (%s)"
	SQL_UPDATE = "UPDATE %s SET %s WHERE %s"
	SQL_DELETE = "DELETE FROM %s WHERE %s"
)

type MysqlController struct {
	db           *sql.DB
	maxOpenConns int
	maxIdleConns int
}

func NewMysqlController() *MysqlController {
	return &MysqlController{maxOpenConns: DefaultOpenConns, maxIdleConns: DefaultIdleConns}
}

func (mc *MysqlController) InitMysql(dbinfo string) error {
	var dbConfig map[string]string
	json.Unmarshal([]byte(dbinfo), &dbConfig)
	if _, ok := dbConfig["user"]; !ok {
		dbConfig["user"] = DefaultUser
	}
	if _, ok := dbConfig["password"]; !ok {
		dbConfig["password"] = ""
	}
	if _, ok := dbConfig["address"]; !ok {
		return ErrMysqlNoHost
	}
	if _, ok := dbConfig["dbname"]; !ok {
		return ErrMysqlNoDBName
	}
	if _, ok := dbConfig["MaxOpenConns"]; !ok {
		dbConfig["MaxOpenConns"] = "200"
	}
	if _, ok := dbConfig["MaxIdleConns"]; !ok {
		dbConfig["MaxIdleConns"] = "100"
	}

	mc.maxOpenConns, _ = strconv.Atoi(dbConfig["MaxOpenConns"])
	mc.maxIdleConns, _ = strconv.Atoi(dbConfig["MaxIdleConns"])
	dbSourceName := dbConfig["user"] + ":" + dbConfig["password"] + "@tcp(" + dbConfig["address"] + ")/" + dbConfig["dbname"] + "?charset=utf8"
	fmt.Println(dbSourceName)
	mc.db, _ = sql.Open("mysql", dbSourceName)
	mc.db.SetMaxOpenConns(mc.maxOpenConns)
	mc.db.SetMaxIdleConns(mc.maxIdleConns)
	return mc.db.Ping()
}

func (mc *MysqlController) Close() {
	if mc.db != nil {
		mc.db.Close()
	}
}

func (mc *MysqlController) checkDB() bool {
	return mc.db != nil
}

// insert
func (mc *MysqlController) Insert(sqlstr string, args ...interface{}) (int64, error) {
	if !mc.checkDB() {
		return 0, ErrMysqlNotInit
	}

	stmtIns, err := mc.db.Prepare(sqlstr)
	if err != nil {
		panic(err.Error())
	}
	defer stmtIns.Close()

	result, err := stmtIns.Exec(args...)
	if err != nil {
		panic(err.Error())
	}
	return result.LastInsertId()
}

// modify or delete
func (mc *MysqlController) Exec(sqlstr string, args ...interface{}) (int64, error) {
	if !mc.checkDB() {
		return 0, ErrMysqlNotInit
	}

	stmtIns, err := mc.db.Prepare(sqlstr)
	if err != nil {
		return 0, err
	}
	defer stmtIns.Close()

	result, err := stmtIns.Exec(args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// query, val type: string
func (mc *MysqlController) FetchRow(sqlstr string, args ...interface{}) (*map[string]string, error) {
	if !mc.checkDB() {
		return nil, ErrMysqlNotInit
	}

	stmtOut, err := mc.db.Prepare(sqlstr)
	if err != nil {
		return nil, err
	}
	defer stmtOut.Close()

	rows, err := stmtOut.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	ret := make(map[string]string, len(scanArgs))

	for i := range values {
		scanArgs[i] = &values[i]
	}
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}
		var value string

		for i, col := range values {
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			ret[columns[i]] = value
		}
		break //get the first row only
	}
	return &ret, nil
}

func (mc *MysqlController) FetchRows(sqlstr string, args ...interface{}) (*[]map[string]string, error) {
	if !mc.checkDB() {
		return nil, ErrMysqlNotInit
	}

	stmtOut, err := mc.db.Prepare(sqlstr)
	if err != nil {
		return nil, err
	}
	defer stmtOut.Close()

	rows, err := stmtOut.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))

	ret := make([]map[string]string, 0)
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}
		var value string
		vmap := make(map[string]string, len(scanArgs))
		for i, col := range values {
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}
			vmap[columns[i]] = value
		}
		ret = append(ret, vmap)
	}
	return &ret, nil
}
