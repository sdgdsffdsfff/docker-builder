package util

import (
	"os"
	"encoding/json"
	"errors"
)

//保存instances 基本信息到本地 json格式
func SaveSnapshot(data interface{}, filePath string) error {

	if filePath == "" {
		return errors.New("save snapshot fail,filePath is empty")
	}
	
	snapshotPath := filePath
	
	out ,err := os.Create(snapshotPath)
	
	if err != nil {
		return err
	}
	
	err = json.NewEncoder(out).Encode(data)
	
	if err != nil {
		return err
	}
	
	out.Close()
	return nil
}