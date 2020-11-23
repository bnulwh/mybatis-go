package orm

import (
	log "github.com/astaxie/beego/logs"
	"io/ioutil"
	"os"
	"strings"
)

type ExecuteFunc func(args ...interface{}) (int64,int64,error)
type QueryRowFunc func(args ...interface{}) (interface{},error)
type QueryRowsFunc func(args ...interface{}) ([]interface{},error)


func LoadSettings(filename string) map[string]string{
	m := LoadProperties(filename)
	em := GetAllEnv()
	for k,v := range m{
		if strings.Compare(v[0,2],"${") == 0{
			v = getRealValue(v,em)
			m[k] = v
		}
	}
	for k,v := range em{
		m[k] = v
	}
	return m
}

func GetAllEnv() map[string]string{
	envMap := map[string]string{}
	for _,envLine := range os.Environ() {
		kv := strings.Split(envLine,"=")
		envMap[kv[0]]=kv[1]
	}
	return envMap
}

func LoadProperties(filename string) map[string]string {
	envMap := map[string]string{}
	body,err := ioutil.ReadFile(filename)
	if err != nil{
		log.Warn("load file %v failed: %v",filename,err)
		return envMap
	}
	for _,line := range strings.Split(string(body),"\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.Contains("!#",line[0,1]) {
			continue
		}
		pos := strings.Index(line,"=")
		if pos <=0 {
			pos = strings.Index(":")
		}
		if pos <=0{
			continue
		}
		key := line[0:pos]
		val := strings.Trim(line[pos+1:],"'\" ")
		envMap[key] = val
	}
	return envMap
}

func getRealValue(v string,em map[string]string) string {
	pos := strings.Index(v,":")
	if pos <0{
		key := v[2:len(v)-1]
		rv,ok := em[key]
		if ok {
			return rv
		}
		return ""
	}
	key := v[2:pos]
	val :=v[pos+1,len(v)-1]
	rv,ok := em[key]
	if ok{
		return rv
	}
	return val
}