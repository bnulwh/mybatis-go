# mybatis-go
a mybatis framework build for golang
# 使用示例（Example）
##生成xml的mapper文件（generate xml mapper files）
使用命令 
```
java -jar mybatis-generator-core-1.3.7.jar -configfile generateConfig.xml
```
生成xml配置文件，如根目录下的`generateConfig.xml`,生成的mapper文件在`resources/mapper`下

##生成generator（可以省略）
使用命令
```
go build -o generator cmd/generator/main.go
```
注意生成的generator的文件权限

##使用generator生成模型和mapper文件（可以省略）
可以使用`./generator -h`查看帮助

##确保配置文件正确

配置文件应至少包含如下内容：
```
spring.datasource.url= jdbc:postgresql://localhost:5432/testdb?useUnicode=true&characterEncoding=utf-8&useSSL=false
 spring.datasource.username= root
 spring.datasource.password= 123456
 mybatis.mapper-locations= resources/mapper
```
----
##开始使用
示例代码见`cmd/postgresdemo/main.go`

###第一步：引入相关包

```
import (
 	log "github.com/astaxie/beego/logs"
 	"github.com/bnulwh/mybatis-go/logger"
 	"github.com/bnulwh/mybatis-go/orm"
 	"github.com/bnulwh/mybatis-go/utils"
 	_ "github.com/lib/pq"
 	"time"
 )
```
----
###第二步：定义模型

```
type UserInfoModel struct {
 	Id          int
 	CreatedBy   string
 	UpdatedBy   string
 	CreateTime  time.Time
 	UpdateTime  time.Time
 	GroupId     int
 	Username    string
 	PassMd5     string
 	Roles       string
 	Description string
 	Avatar      string
 }
```
----
###第三步：定义Dao/Mapper

注意：必须继承`orm.BaseMapper`

```
type UserInfoModelMapper struct {
 	orm.BaseMapper
 	DeleteByPrimaryKey orm.ExecuteFunc
 	Insert             orm.ExecuteFunc
 	UpdateByPrimaryKey orm.ExecuteFunc
 	SelectByPrimaryKey orm.QueryRowsFunc
 	SelectAll          orm.QueryRowsFunc
 }
```
----
###第四步: 初始化ORM框架

```
func init() {
 	orm.Initialize("application-pg.properties")
 	orm.RegisterModel(new(UserInfoModel))
 	orm.RegisterMapper(new(UserInfoModelMapper))
 }
```
----
###最后：使用ORM

```
func main() {
 	defer orm.Close()
 	mp := orm.NewMapper("UserInfoModelMapper").(UserInfoModelMapper)
 	rs, err := mp.SelectAll()
 	if err != nil {
 		log.Error("select failed: %v", err)
 	} else {
 		for _, row := range rs {
 			log.Info("row: %v", utils.ToJson(row))
 		}
 	}
 }
```
------
#路线图
* 1.支持postgres数据库使用，已经测试通过
* 2.加入对mysql数据库的支持，开发中
* 3.加入对sqlite数据库的支持
* 4.多数据源的支持
* 5.其他改进和优化