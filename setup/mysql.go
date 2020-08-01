package setup

import (
	"bytes"
	"fmt"
	"gitee.com/kelvins-io/common/env"
	"gitee.com/kelvins-io/kelvins/config/setting"
	"net/url"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"xorm.io/xorm"
	xormLog "xorm.io/xorm/log"
)

// NewMySQL returns *gorm.DB instance.
func NewMySQLWithGORM(mysqlSetting *setting.MysqlSettingS) (*gorm.DB, error) {
	if mysqlSetting == nil {
		return nil, fmt.Errorf("mysqlSetting is nil")
	}
	if mysqlSetting.UserName == "" {
		return nil, fmt.Errorf("lack of mysqlSetting.UserName")
	}
	if mysqlSetting.Password == "" {
		return nil, fmt.Errorf("lack of mysqlSetting.Password")
	}
	if mysqlSetting.Host == "" {
		return nil, fmt.Errorf("lack of mysqlSetting.Host")
	}
	if mysqlSetting.DBName == "" {
		return nil, fmt.Errorf("lack of mysqlSetting.DBName")
	}
	if mysqlSetting.Charset == "" {
		return nil, fmt.Errorf("lack of mysqlSetting.Charset")
	}
	if mysqlSetting.PoolNum == 0 {
		return nil, fmt.Errorf("lack of mysqlSetting.PoolNum")
	}

	var buf bytes.Buffer
	buf.WriteString(mysqlSetting.UserName)
	buf.WriteString(":")
	buf.WriteString(mysqlSetting.Password)
	buf.WriteString("@tcp(")
	buf.WriteString(mysqlSetting.Host)
	buf.WriteString(")/")
	buf.WriteString(mysqlSetting.DBName)
	buf.WriteString("?charset=")
	buf.WriteString(mysqlSetting.Charset)
	buf.WriteString("&parseTime=" + strconv.FormatBool(mysqlSetting.ParseTime))
	buf.WriteString("&multiStatements=" + strconv.FormatBool(mysqlSetting.MultiStatements))
	if mysqlSetting.Loc == "" {
		buf.WriteString("&loc=Local")
	} else {
		buf.WriteString("&loc=" + url.QueryEscape(mysqlSetting.Loc))
	}

	db, err := gorm.Open("mysql", buf.String())
	if err != nil {
		return nil, err
	}
	if env.IsDevMode() {
		db.LogMode(true)
	}

	db.DB().SetConnMaxLifetime(30 * time.Second)
	if mysqlSetting.ConnMaxLifeSecond > 0 {
		db.DB().SetConnMaxLifetime(time.Duration(mysqlSetting.ConnMaxLifeSecond) * time.Second)
	}

	maxIdle := 10
	maxOpen := 10
	if mysqlSetting.MaxOpen > 0 && mysqlSetting.MaxIdle > 0 {
		maxIdle = mysqlSetting.MaxIdle
		maxOpen = mysqlSetting.MaxOpen
	}
	db.DB().SetMaxIdleConns(maxIdle)
	db.DB().SetMaxOpenConns(maxOpen)

	return db, nil
}

// NewMySQL returns *xorm.DB instance.
func NewMySQLWithXORM(mysqlSetting *setting.MysqlSettingS) (xorm.EngineInterface, error) {
	if mysqlSetting == nil {
		return nil, fmt.Errorf("mysqlSetting is nil")
	}
	if mysqlSetting.UserName == "" {
		return nil, fmt.Errorf("lack of mysqlSetting.UserName")
	}
	if mysqlSetting.Password == "" {
		return nil, fmt.Errorf("lack of mysqlSetting.Password")
	}
	if mysqlSetting.Host == "" {
		return nil, fmt.Errorf("lack of mysqlSetting.Host")
	}
	if mysqlSetting.DBName == "" {
		return nil, fmt.Errorf("lack of mysqlSetting.DBName")
	}
	if mysqlSetting.Charset == "" {
		return nil, fmt.Errorf("lack of mysqlSetting.Charset")
	}
	if mysqlSetting.PoolNum == 0 {
		return nil, fmt.Errorf("lack of mysqlSetting.PoolNum")
	}

	var buf bytes.Buffer
	buf.WriteString(mysqlSetting.UserName)
	buf.WriteString(":")
	buf.WriteString(mysqlSetting.Password)
	buf.WriteString("@tcp(")
	buf.WriteString(mysqlSetting.Host)
	buf.WriteString(")/")
	buf.WriteString(mysqlSetting.DBName)
	buf.WriteString("?charset=")
	buf.WriteString(mysqlSetting.Charset)
	buf.WriteString("&parseTime=" + strconv.FormatBool(mysqlSetting.ParseTime))
	buf.WriteString("&multiStatements=" + strconv.FormatBool(mysqlSetting.MultiStatements))
	if mysqlSetting.Loc == "" {
		buf.WriteString("&loc=Local")
	} else {
		buf.WriteString("&loc=" + url.QueryEscape(mysqlSetting.Loc))
	}

	engine, err := xorm.NewEngine("mysql", buf.String())
	if err != nil {
		return nil, err
	}
	if env.IsDevMode() {
		engine.SetLogLevel(xormLog.LOG_DEBUG)
		engine.ShowSQL(true)
	}

	engine.SetConnMaxLifetime(30 * time.Second)
	if mysqlSetting.ConnMaxLifeSecond > 0 {
		engine.SetConnMaxLifetime(time.Duration(mysqlSetting.ConnMaxLifeSecond) * time.Second)
	}

	maxIdle := 10
	maxOpen := 10
	if mysqlSetting.MaxOpen > 0 && mysqlSetting.MaxIdle > 0 {
		maxIdle = mysqlSetting.MaxIdle
		maxOpen = mysqlSetting.MaxOpen
	}

	engine.SetMaxIdleConns(maxIdle)
	engine.SetMaxOpenConns(maxOpen)

	return engine, nil
}

// SetGORMCreateCallback is set create callback
func SetGORMCreateCallback(db *gorm.DB, callback func(scope *gorm.Scope)) {
	db.Callback().Create().Replace("gorm:update_time_stamp", callback)
}

// SetGORMUpdateCallback is set update callback
func SetGORMUpdateCallback(db *gorm.DB, callback func(scope *gorm.Scope)) {
	db.Callback().Update().Replace("gorm:update_time_stamp", callback)
}

// SetGORMDeleteCallback is set delete callback
func SetGORMDeleteCallback(db *gorm.DB, callback func(scope *gorm.Scope)) {
	db.Callback().Delete().Replace("gorm:delete", callback)
}
