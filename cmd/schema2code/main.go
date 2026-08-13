package main

import (
	"flag"
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/orm"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func init() {
	//log.SetLevel(log.InfoLevel)
	log.ConfigLocalFileSystemLogger("logs", "schema2code")
	orm.SetLogger(log.StandardLogger())
}

func main() {
	var dbType, host, user, pwd, dbName, dir, prefix, tables string
	var port int
	flag.StringVar(&dbType, "type", "mysql", "数据库类型: mysql/postgres/sqlite")
	flag.StringVar(&host, "host", "localhost", "数据库地址: localhost")
	flag.IntVar(&port, "port", 3306, "数据库端口")
	flag.StringVar(&user, "username", "", "用户名(sqlite 无需填写)")
	flag.StringVar(&pwd, "password", "", "密码(sqlite 无需填写)")
	flag.StringVar(&dbName, "db", "", "数据库名(sqlite 为 .db 文件路径)")
	flag.StringVar(&dir, "output", "temp", "保存路径")
	flag.StringVar(&prefix, "prefix", "", "表名前缀")
	flag.StringVar(&tables, "tables", "", "数据库表,分隔符用英文逗号 ','")
	flag.Parse()
	if dbName == "" || (dbType != "sqlite" && dbType != "sqlite3" && (user == "" || pwd == "")) {
		flag.Usage()
		return
	}
	orm.InitializeDatabase(dbType, host, port, user, pwd, dbName)
	defer orm.Close()
	orm.SchemaToCode(dir, prefix, tables)
}
