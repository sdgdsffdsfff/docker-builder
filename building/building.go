package building

import (
	steno "github.com/cloudfoundry/gosteno"
	"builder/config"
	"errors"
	"fmt"
)

//创建一个打包执行对象
func NewBuilding(conf *config.Config) *BuildingHandler {
	
	registry := NewRegistry(conf.Builder.SnapshotPath)
	
	return &BuildingHandler{
		confg:			conf,
		logger:			steno.NewLogger("builder"),
		registry:       registry,	
	}
}

//启动 打包处理对象
func (b *BuildingHandler) Start() {
	//加载本地缓存的snapshot文件
	b.registry.loadSnapshot()
	//启动多个线程处理数据
	
	for v:=1;v<=b.confg.Builder.WorkThreadSize ;v++ {
		job := NewJob(b.registry, v, b.confg)
		go job.Start()
	}
}

//注册需要打包的任务到队列
func (b *BuildingHandler) Register(building *Building) error {
	
	err := b.validate(building)
	if err != nil {
		return errors.New(fmt.Sprintf("Register Building Task fail, %s", err.Error() ) )	
	}
	
	err = b.registry.Add(building)
	
	if err != nil {
		return errors.New(fmt.Sprintf("Register Building Task fail, %s", err.Error() ) )	
	}
	return nil
}

//返回队列中所有的数据
func (b *BuildingHandler) GetAllBuildings()map[string]map[string]*Building{
	b.registry.lock.Lock()
	defer b.registry.lock.Unlock()
	
	result := make(map[string]map[string]*Building)
	wait := make(map[string]*Building)
	run  := make(map[string]*Building)
	fail := make(map[string]*Building)
	
	result["wait"] = wait
	result["running"] = run
	result["fail"] = fail
	
	for key,v := range b.registry.builds {
		if v.Status == BUILDING_STATUS_WAITE {
			wait[key] = v
		}else if v.Status == BUILDING_STATUS_RUNNERING	{
			run[key] = v
		}else if v.Status == BUILDING_STATUS_FAIL {
			fail[key] = v
		}
	}
	return result
}

//根据guid查询队列中的数据
func (b *BuildingHandler) SearchBUildings(appguid string) *Building {
	b.registry.lock.Lock()
	defer b.registry.lock.Unlock()
	
	for _,v := range b.registry.builds {
		if v.AppGuid == appguid {
			return v	
		}
	}
	return nil
}
//检测打包任务的参数是否合法
func (b *BuildingHandler) validate(building *Building) error {
	if building == nil {
		return errors.New(" building is nil")
	}
	
	if building.AppGuid == "" {
		return errors.New(" app_guid is empty")
	}
	
	if building.AppName == "" {
		return errors.New(" app_name is empty")	
	}
	
	if building.Language == "" {
		return errors.New(" language is empty")	
	}
	
	if building.ApplicationServer == "" {
		return errors.New(" applicationServer is empty")		
	}
	
	if building.RuntimeEnvironment == "" {
		return errors.New(" runtimeEnvironment is empty")	
	}
	
	return nil
}
