package util

import (
	"os"
	"io"
	"archive/zip"
	"path/filepath"
	"errors"
	"fmt"
)

// 解压文件, source :待解压的文件, dest: 减压后存放的目录
func UnzipFile(source string , dest string) error {
	unzipFile, err := zip.OpenReader(source)
	if err != nil {
		return errors.New(fmt.Sprintf("Unzipfile OpenReader source:%s, fail:%s", source, err) )
	}
	defer unzipFile.Close()
	for _,f := range unzipFile.File {
		rc ,err := f.Open()
		if err != nil {
			return	errors.New(fmt.Sprintf("Unzipfile openFile source:%s, fail:%s", source, err) )
		}
		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0755)	
		}else {
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE | os.O_TRUNC, 0755)	
			if err != nil {
				rc.Close()
				return	errors.New(fmt.Sprintf("Unzipfile openFile source:%s, fail:%s", source, err) )
			}
			_, err = io.Copy(f ,rc)
			if err != nil  && err != io.EOF {
				f.Close()
				rc.Close()
				return	errors.New(fmt.Sprintf("Unzipfile openFile source:%s, fail:%s", source, err) )
			}
			f.Close()
			rc.Close()
		}
	}
	return nil
}