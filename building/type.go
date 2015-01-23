package building

import (
	"sync"
	steno "github.com/cloudfoundry/gosteno"
	"builder/config"
	"time"
)

const(
		//等待打包
		BUILDING_STATUS_WAITE = 1
		//正在打包
		BUILDING_STATUS_RUNNERING = 2
		//打包成功
		BUILDING_STATUS_SUCCESS = 3
		//打包失败
		BUILDING_STATUS_FAIL = -1
	)

type BuildingRegistry struct {
	lock 				sync.Mutex
	logger				*steno.Logger
	builds				map[string]*Building
	snapshotPath		string
}

//处理打包的对象
type BuildingHandler struct {
	logger				*steno.Logger
	registry			*BuildingRegistry
	confg				*config.Config
}

type Building struct {
	AppGuid				string //应用guid
	StepsId				string //步骤ID
	AppName				string //应用名称
	Language			string //语言类型(java/php/)
	ApplicationServer	string //应用服务器信息(比如:tomcat-6.0.2/apache-1.2.1)
	RuntimeEnvironment	string //运行环境(比如:jdk-1.6.11/ php-7.1)
	CreateTime			time.Time //任务创建时间
	StartTime			time.Time //任务开始执行时间
	EndTime				time.Time //任务执行完成时间
	Status				int
	BuildMessage		string //记录错误日志
	BuildCount			int //打包重复执行此次
}