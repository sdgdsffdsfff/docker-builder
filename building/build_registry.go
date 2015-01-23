package building

import (
	steno "github.com/cloudfoundry/gosteno"
	"builder/util"
	"encoding/json"
	"io/ioutil"
	"path"
	"time"
	"os"
)

const(
		SNAPSHOT_NAME = "building_registry.json"
	)

func NewRegistry(path string) *BuildingRegistry {
		
	return &BuildingRegistry{
		logger:				steno.NewLogger("builder"),
		builds:				make(map[string]*Building),	
		snapshotPath:		path,
	}
}

func (r *BuildingRegistry) Add(building *Building) error {
	r.logger.Infof("buldingRegistry Add,guid:%s", building.AppGuid)
	r.lock.Lock()
	defer r.lock.Unlock()
	
	building.CreateTime = time.Now()
	building.Status = BUILDING_STATUS_WAITE
	building.BuildCount = 0
	r.builds[building.AppGuid] = building
	
	err := r.saveSnapshot()
	
	return err
}

func (r *BuildingRegistry) Remove(appguid string) {
	r.logger.Infof("buildingRegistry Remove,guid:%s", appguid)
	r.lock.Lock()
	defer r.lock.Unlock()
	
	_,found := r.builds[appguid]
	if found {
		delete(r.builds, appguid)
	}
	err := r.saveSnapshot()
	if err != nil {
		r.logger.Errorf("buildingRegistry Remove,guid:%s, fail:%s", appguid, err.Error() )	
	}
}

//获取一个等待打包的任务
func (r *BuildingRegistry) GetWaitTask() *Building{
	r.logger.Infof("buildingRegistry GetWaitTask......................")
	r.lock.Lock()
	defer r.lock.Unlock()
	
	if len(r.builds) == 0 {
		return nil
	}
	
	for k,v := range r.builds {
		if v.Status == BUILDING_STATUS_WAITE {
			r.logger.Infof("buildingRegistry GetWaitTask,return guid:%s", k)
			v.StartTime = time.Now()
			r.updateTaskStatus(v , BUILDING_STATUS_RUNNERING, "")
			return v	
		}
	}
	return nil
}

//修改打包任务状态
func (r *BuildingRegistry) updateTaskStatus(task *Building , status int, msg string) {
	
	task.Status = status
	task.BuildMessage = msg
	//save snapshot
	err := r.saveSnapshot()
	if err != nil {
		r.logger.Errorf("buildingRegistry GetWaitTask saveSnapshot fail:%s", err.Error() )	
	}
}

func (r *BuildingRegistry) loadSnapshot() {
	r.logger.Infof("loadSnapshot ........................")
	snapshotPath := r.path()
	_, err := os.Stat(snapshotPath)
	if err != nil {
		r.logger.Infof("Snapshot Load fail path:%s ,msg: %s", snapshotPath, err.Error() )
		return
	}
	
	bytes , err := ioutil.ReadFile(snapshotPath)
	if err != nil {
		r.logger.Errorf("Snapshot Load readFile:%s fail, %s", snapshotPath, err.Error() )
		return
	}
	
	loads := make(map[string]*Building)
	
	err = json.Unmarshal(bytes, &loads)
	if err != nil {
		r.logger.Errorf("Snapshot Load readFile:%s fail, %s", snapshotPath, err.Error() )
		return
	}
	
	r.lock.Lock()
	defer r.lock.Unlock()
	
	r.builds = loads
	r.logger.Infof("loadSnapshot ........................success,length:%s", len(r.builds))
}

func (r *BuildingRegistry) saveSnapshot() error {
	r.logger.Infof("save Snapshot Building Registry data..................")
	err := util.SaveSnapshot(r.builds, r.path())
	
	return err
}

func (r *BuildingRegistry) path() string {
	return path.Join(r.snapshotPath, SNAPSHOT_NAME) 	
}

