package main

import (
	"flag"
	log "github.com/bnulwh/logrus"
	"github.com/bnulwh/mybatis-go/orm"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func init() {
	//log.SetLevel(log.InfoLevel)
	log.ConfigLocalFileSystemLogger("logs", "schema2code")
}

func main() {
	var dbType, host, user, pwd, dbName, dir string
	var port int
	flag.StringVar(&dbType, "type", "postgres", "database type: mysql/postgres")
	flag.StringVar(&host, "host", "localhost", "database address,default: localhost")
	flag.IntVar(&port, "port", 5432, "database port")
	flag.StringVar(&user, "username", "postgres", "database username")
	flag.StringVar(&pwd, "password", "123456", "database password")
	flag.StringVar(&dbName, "db", "harbor_clair", "database")
	flag.StringVar(&dir, "output", "temp", "saving folder")
	flag.Parse()
	if user == "" || pwd == "" || dbName == "" {
		flag.Usage()
		return
	}
	orm.InitializeDatabase(dbType, host, port, user, pwd, dbName)
	defer orm.Close()
	orm.SchemaToCode(dir)
}
