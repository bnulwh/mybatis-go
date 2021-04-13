# mybatis-go
a mybatis orm framework build for golang
# 使用示例（Example）
## 生成xml的mapper文件（generate xml mapper files）
使用命令 
```
java -jar mybatis-generator-core-1.3.7.jar -configfile generateConfig.xml
```
生成xml配置文件，如根目录下的`generateConfig.xml`,生成的mapper文件在`resources/mapper`下

## 生成generator（可以省略）
使用命令
```
go build -o generator cmd/generator/main.go
```
注意生成的generator的文件权限

## 使用generator生成模型和mapper文件（可以省略）
可以使用`./generator -h`查看帮助
```
Usage of ./generator:
  -d string
    	saving directory,default: temp (default "temp")
  -m string
    	sql mapper file directory,default: resources/mapper (default "resources/mapper")
  -p string
    	package name,default: temp (default "temp")
```
生成的模型文件示例如下:
```
package temp

import(
	"github.com/bnulwh/mybatis-go/orm"
	"time"
)

type UserInfoModel struct{
	Id 	int
	CreatedBy 	string
	UpdatedBy 	string
	CreateTime 	time.Time
	UpdateTime 	time.Time
	GroupId 	int
	Username 	string
	PassMd5 	string
	Roles 	string
	Description 	string
	Avatar 	string
}

func init(){
	orm.RegisterModel(new(UserInfoModel))
}
```
生成的dao/mapper文件示例如下:
```
package temp

import (
	"github.com/bnulwh/mybatis-go/orm"
) 

type UserInfoModelMapper struct {
	orm.BaseMapper
	DeleteByPrimaryKey func(int32) (int64, error) `args:id`
	Insert             func(UserInfoModel) (int64, error)
	UpdateByPrimaryKey func(UserInfoModel) (int64, error)
	SelectByPrimaryKey func(int32) ([]UserInfoModel, error) `args:id`
	SelectAll          func() ([]UserInfoModel, error)
}

func init() {
	orm.RegisterMapper(new(UserInfoModelMapper))
}

```
注意：
* orm.BaseMapper为dao/mapper的父类
* Mapper struct的各个func类型属性会在生成时赋值，对应XML中的SQL函数
* func类型的返回值个数只能有两个，SELECT类型的为返回的查询结果和error类型，INSERT/UPDATE/DELETE类型的返回值要求为int64和error类型
* func类型属性的tag字段可以用来标示输入参数的名称

## 确保配置文件正确

配置文件应至少包含如下内容：
```
spring.datasource.url= jdbc:postgresql://localhost:5432/testdb?useUnicode=true&characterEncoding=utf-8&useSSL=false
spring.datasource.username= root
spring.datasource.password= 123456
mybatis.mapper-locations= resources/mapper
```
----
## 开始使用
示例代码见`cmd/postgresdemo/main.go`

### 第一步：引入相关包

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
### 第二步：定义模型

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
### 第三步：定义Dao/Mapper

注意：必须继承`orm.BaseMapper`

```
type UserInfoModelMapper struct {
 	orm.BaseMapper
	DeleteByPrimaryKey func(int32) (int64, error)
	Insert             func(UserInfoModel) (int64, error)
	UpdateByPrimaryKey func(UserInfoModel) (int64, error)
	SelectByPrimaryKey func(int32) ([]UserInfoModel, error)
	SelectAll          func() ([]UserInfoModel, error)
 }
```
----
### 第四步: 初始化ORM框架

```
func init() {
 	orm.Initialize("application-pg.properties")
 	orm.RegisterModel(new(UserInfoModel))
 	orm.RegisterMapper(new(UserInfoModelMapper))
 }
```
----
### 最后：使用ORM

```
func main() {
 	defer orm.Close()
 	mp := orm.NewMapper("UserInfoModelMapper").(UserInfoModelMapper)
 	rs, err := mp.SelectAll()
 	if err != nil {
 		log.Error("select failed: %v", err)
 	} else {
 		for _, row := range rs {
 			log.Info("row: %v", types.ToJson(row))
 		}
 	}
 }
```
注意：
* `orm.NewMapper("UserInfoModelMapper").(UserInfoModelMapper)`用来创建dao/mapper的实体对象
* `orm.NewMapper("UserInfoModelMapper")`创建的对象必须先注册`orm.RegisterMapper(new(UserInfoModelMapper))`
* 创建后的对象可以使用相关的方法操作数据库
* `orm.RegisterModel(new(UserInfoModel))`用于注册model类，注册后的类在调用dao/mapper的函数时可以自动创建并填充值

# 路线图
* 1.支持postgres数据库使用，已经测试通过
* 2.加入对mysql数据库的支持，开发中
* 3.加入对sqlite数据库的支持
* 4.多数据源的支持
* 5.其他改进和优化