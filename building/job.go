package building

import (
	steno "github.com/cloudfoundry/gosteno"
	"icode.jd.com/cdlxyong/go-dockerclient"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"builder/config"
	"builder/util"
	"strconv"
	"time"
	"strings"
	"fmt"
	"path"
	"net/http"
    "io/ioutil"
    "io"
    "os"
    "bytes"
    "errors"
    "bufio"
	"unicode"
)

const(
		DockerFileName = "Dockerfile"
	)
type Job struct {
	logger				*steno.Logger
	registry			*BuildingRegistry
	name				string
	conf				*config.Config
	jss					*util.JssUtil
	buildLogJss			*util.JssUtil
	replaceCmd			map[string]string
}

func NewJob(registry *BuildingRegistry, index int, conf *config.Config) *Job {
	
	//初始化替换Dockerfile中命令的数据结构
	replaceCmd := make(map[string]string)
	replaceCmd["app_package_to_container"] = "ADD ./app		/export/Data"
	
	return &Job{
			logger:			steno.NewLogger("builder"),
			registry:		registry,
			name:			"job-"+strconv.Itoa(index),
			conf:			conf,
			jss:			util.NewJssUtil(conf),
			replaceCmd:		replaceCmd,
		}
}

func (j *Job) Start() {
	
	j.logger.Infof("%s starting", j.name)
	for{
		task := j.registry.GetWaitTask()
		if task != nil {
			j.build(task)
		}else {
			j.logger.Infof("%s no task wait execute,sleep 1 second", j.name)
			time.Sleep(time.Duration(j.conf.Builder.JobIntervalSecond) * time.Second)
		}
		
	}
}

//打包
func (j *Job) build(task *Building) {
	j.logger.Infof("%s,starting build,guid:%s,appname:%s,language:%s",j.name, task.AppGuid, task.AppName, task.Language)
	start := time.Now()
	
	//init build 
	buildlogfilepath, buildtmp, err := j.initBuild(task)
	if err != nil {
		j.complieBuildFail(task, err, "", buildtmp, "")
		return
	}
	
	dockerfileName := j.dockerfileName(task)
	dockerfilePath, err := j.downloadDockerfile(dockerfileName)
	if err != nil {
		j.complieBuildFail(task, err, "", buildtmp, "")
		return
	}
	//download app package
	err = j.downloadAppPackage(task, buildtmp)
	if err != nil {
		j.complieBuildFail(task, err, "", buildtmp, "")	
		return
	}
	//parse dockerfile
	err = j.parseDockerfile(dockerfilePath, buildtmp)
	if err != nil {
		j.complieBuildFail(task, err , "", buildtmp, "")	
		return
	}
	
	//开始编译文件
	imageId , err := j.buildDockerfile(buildlogfilepath, buildtmp, task)
	if err != nil {
		j.complieBuildFail(task, err , buildlogfilepath, buildtmp, imageId)	
		return
	}
	
	pushCount :=3
	for v:=0;v<pushCount;v++ {
		//push image
		err = j.pushImage(imageId, buildlogfilepath)
		if err != nil {
			j.logger.Warnf("push image:%s, fail:%s, try count:%s ", imageId, err, v)
			continue
		}
		break
	}
	
	if err != nil {
		j.complieBuildFail(task, err , buildlogfilepath, buildtmp, imageId)	
		return
	}
	
	j.complieBuildSuccess(task, buildlogfilepath, buildtmp, imageId)
	end := time.Now()
	j.logger.Infof("%s,success build,guid:%s,appname:%s,language:%s, timer:%s", j.name, task.AppGuid, task.AppName, task.Language, end.Sub(start).Seconds() )
}

//开始编译dockerfile
func (j *Job) buildDockerfile(logPath string, buildDir string, task *Building) (string,error) {
	dockerClient , err := docker.NewClient(j.conf.Builder.DockerUrl)
	if err != nil {
		return "", err	
	}
	
	guid, err := util.GetGuid()
	if err != nil {
		return "", err
	}
	
	imageId := j.conf.Builder.DockerRegistry+"/"+guid
	j.logger.Infof("begin build Dockerfile imageId:%s", imageId)
	logfile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE | os.O_TRUNC, 0755)
	if err != nil {
		return "", err	
	}
	
	defer logfile.Close()
	
	imageOptions := docker.BuildImageOptions{
		Name:		imageId,
		NoCache:	true,
		SuppressOutput:	false,
		RmTmpContainer:	true,
		ForceRmTmpContainer: true,
		OutputStream:	logfile,
		RawJSONStream:	true,
		ContextDir:		buildDir,
	}
	
	err = dockerClient.BuildImage(imageOptions)
	
	if err != nil {
		return "", err	
	}
	j.logger.Infof("build Dockerfile imageId:%s  success", imageId)
	return imageId, nil
}

//解析dockerFile 文件,负责在dockerfile中添加ADD app to path 的命令,同时将dockerfile拷贝到临时打包目录中
func (j *Job) parseDockerfile(cacheDockerfile string, buildtmp string) error {
	f , err := os.Open(cacheDockerfile)
	if err != nil {
		return errors.New(fmt.Sprintf("parseDockerfile :%s, fail:%s ", cacheDockerfile, err) )
	}
	defer f.Close()
	tmpdockerfile := path.Join(buildtmp, DockerFileName)
	dockerfile, err := os.OpenFile(tmpdockerfile, os.O_WRONLY|os.O_CREATE | os.O_TRUNC, 0755)	
	if err != nil {
		return errors.New(fmt.Sprintf("parseDockerfile and create Dockerfile:%s in tmp directory fail:%s ", tmpdockerfile, err) )
	}
	
	defer dockerfile.Close()
	
	//开始解析文件
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimLeftFunc(scanner.Text(), unicode.IsSpace)
		
		//判断命令是否要替换
		val,found := j.replaceCmd[line]
		if found {
			j.logger.Infof("parse Dockerfile:%s, and replace cmd:[%s] to: [%s]", cacheDockerfile, line, val)
			line = val
		}
		_,err := dockerfile.WriteString(line+"\n")
		if err != nil {
			return errors.New(fmt.Sprintf("parseDockerfile and writeString to Dockerfile fail:%s ", err) )
		}
	}
	j.logger.Infof("parse docker file success, cache:%s, tmp : %s", cacheDockerfile, tmpdockerfile)
	return nil
}

//根据task参数生成对应的dockerfile 名称(规则: language_runtime_service 比如:java_jdk1.6.25_tomcat_6.0.3)
func (j *Job) dockerfileName(task *Building) string {
	return fmt.Sprintf("%s_%s_%s", strings.ToLower(task.Language), strings.ToLower(task.RuntimeEnvironment), strings.ToLower(task.ApplicationServer) )
}

//根据dockerfile 名称组装dockerfile的缓存路径
func (j *Job) dockerfileCatch(dockerfileName string) string {
	return path.Join(j.conf.Builder.DockerFilePath, dockerfileName)
}

//判断本地是否存在dockerfile文件如果不存在则从packageserver上下载对应的dockerfile
func (j *Job) downloadDockerfile(dockerfileName string) (string,error) {
	dockerfile := j.dockerfileCatch(dockerfileName)
	j.logger.Infof("downloadDockerfile cache path: %s", dockerfile)
	
	_, err := os.Stat(dockerfile)
	if err == nil {
		j.logger.Infof("no downloadDockerfile cache path: %s, cache exist", dockerfile)
		return dockerfile, nil
	}
	
	url := fmt.Sprintf("%s%s", j.conf.Builder.DockerFileRemotePath, dockerfileName)
	j.logger.Infof("begin downloadDockerfile remout url: %s", url)
	
	file , err := os.Create(dockerfile)
	if err != nil {
		j.logger.Errorf("downloadDockerfile cache path: %s,create file fail:%s", dockerfile, err)
		return "",errors.New(fmt.Sprintf("downloadDockerfile cache path: %s,create file fail:%s", dockerfile, err) )
	}
	defer file.Close()
	resp, err := http.Get(url)
	if err != nil {
		j.logger.Errorf("downloadDockerfile url: %s,fail:%s", url, err)
		return "", errors.New(fmt.Sprintf("downloadDockerfile url: %s,fail:%s", url, err) )
	}
	
	defer resp.Body.Close()
	
	pix , err := ioutil.ReadAll(resp.Body)
	if err != nil {
		j.logger.Errorf("downloadDockerfile url: %s,read body fail:%s", url, err)
		return "", errors.New(fmt.Sprintf("downloadDockerfile url: %s,read body fail:%s", url, err) )
	}
	
	n, err := io.Copy(file, bytes.NewReader(pix))
	
	if err != nil {
		j.logger.Errorf("downloadDockerfile url: %s,copy file fail:%s", url, err)
		return "",errors.New(fmt.Sprintf("downloadDockerfile url: %s,copy file fail:%s", url, err) )
	}
	if n == 0 {
		j.logger.Errorf("downloadDockerfile url: %s,copy file length:0", url)
		return "",errors.New("copy file length:0")
	}
	
	return dockerfile,nil
}

//初始化打包环境,创建打包的临时目录,生成打包的日志文件
//return buildlogfile,buildtmp,err
func (j *Job) initBuild(task *Building) (string, string, error) {
	
	buildTmp := path.Join(j.conf.Builder.BuildPath, task.AppGuid)
	
	//创建打包临时目录
	dir , err := os.Stat(buildTmp)
	if err != nil || !dir.IsDir(){
		err := os.MkdirAll(buildTmp, 0777)
		if err != nil {
			j.logger.Errorf("init build and create tmp build dir fail:%s", err)
			return "","",errors.New(fmt.Sprintf("init build and create tmp build dir fail:%s", err) )
		}
	}
	
	//创建打包的日志文件
	guid ,err := util.GetGuid()
	if err != nil {
		j.logger.Errorf("init build and get build_log_file guid  fail:%s", err)
		return "","",errors.New(fmt.Sprintf("init build and get build_log_file guid  fail:%s", err) )
	}
	buildLogfilePath := path.Join(buildTmp, guid)
	buildLogfile, err := os.Create(buildLogfilePath)
	if err != nil {
		j.logger.Errorf("init build and create buildlogfile fail:%s", err)
		return "","",errors.New(fmt.Sprintf("init build and create buildlogfile fail:%s", err) )
	}
	defer buildLogfile.Close()
	
	j.logger.Infof("init build success, logfile:%s, buildtmp:%s", buildLogfilePath, buildTmp)
	return buildLogfilePath, buildTmp, nil
}

//下载package
func (j *Job) downloadAppPackage(task *Building, buildtmp string) error {
	
	appzipPath := path.Join(buildtmp, task.AppGuid)
	
	downTry := 3
	var err error
	for v:=0;v<downTry; v++ {
		err := j.jss.Download(task.AppGuid, appzipPath)
		if err != nil {
			j.logger.Warnf("download app Package:%s from jss fail,try:%s", appzipPath, v)
			os.RemoveAll(appzipPath)
			continue	
		}
	}
	if err != nil {
		return err	
	}
	zip, err := os.Stat(appzipPath)
	
	if err != nil {
		return errors.New(fmt.Sprintf("downloadAppPackage fail , appZip:%s, is not exists", appzipPath) )	
	}
	if zip.Size() <=0 {
		return errors.New(fmt.Sprintf("downloadAppPackage fail , appZip:%s, is empty", appzipPath) )
	}
	
	//创建unzip dir
	appPath := path.Join(buildtmp, "app")
	err = os.MkdirAll(appPath, 0777)
	if err != nil {
		return errors.New(fmt.Sprintf("downloadApppackage and create unzip directory fail:%s", err) )
	}
	
	//unzip 
	err = util.UnzipFile(appzipPath, appPath)
	if err != nil {
		return err
	}
	return nil
}

//build fail
func (j *Job) complieBuildFail(task *Building, err error, logfile string, tmpdir string, imageId string) {
	j.logger.Errorf("complieBuildFail app:%s,name:%s build fail:%s", task.AppGuid, task.AppName, err.Error() )
	
	//修改打包任务状态
	j.registry.updateTaskStatus(task, BUILDING_STATUS_FAIL, err.Error() )
	
	//上传打包日志到 jss
	if logfile != "" {
		j.jss.Upload(logfile)
	}
	
	task.EndTime = time.Now()
	j.updateDb(task, logfile, err.Error() )
	
	//清空打包的临时文件夹
	j.cleanTmpDir(tmpdir)
	
	//清楚本地image
	j.removeTmpImage(imageId)
	
}

//build success
func (j *Job) complieBuildSuccess(task *Building, logfile string, tmpdir string, imageId string) {
	j.logger.Infof("complieBuildSuccess app:%s,name:%s build success", task.AppGuid, task.AppName)
	
	//删除内存中的打包任务
	j.registry.Remove(task.AppGuid)

	//上传打包日志到 jss
	j.jss.Upload(logfile)
	
	task.Status = BUILDING_STATUS_SUCCESS
	task.EndTime = time.Now()
	j.updateDb(task, logfile, "")
	
	//清空打包的临时文件夹
	j.cleanTmpDir(tmpdir)
	
	//删除本地image
	j.removeTmpImage(imageId)
}

//删除打包的临时目录
func (j *Job) cleanTmpDir(tmpdir string) {
	if _, err := os.Stat(tmpdir); err == nil {
        j.logger.Infof("remove bulid tmp dir:%s",tmpdir)
        os.RemoveAll(tmpdir)
    }
}

//删除image
func (j *Job) removeTmpImage(imageId string) {
	dockerClient , err := docker.NewClient(j.conf.Builder.DockerUrl)
	if err != nil {
		j.logger.Errorf("remove tmp image:%s fail, create dockerClient fail.%s", imageId, err)
	}
	
	err = dockerClient.RemoveImage(imageId)
	if err != nil {
		j.logger.Errorf("remove tmp image:%s fail.%s", imageId, err)	
	}
	
	j.logger.Infof("remove tmp image:%s success", imageId)
}

//上传image
func (j *Job) pushImage(imageId string, logfile string) error {
	j.logger.Infof("begin push image to private docker registry imageId:%s", imageId)
	start := time.Now()
	
	dockerClient , err := docker.NewClient(j.conf.Builder.DockerUrl)
	if err != nil {
		return errors.New(fmt.Sprintf("pushImage image:%s fail, create dockerClient fail.%s", imageId, err) )
	}
	
	logWrite, err := os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE | os.O_TRUNC, 0755)
	if err != nil {
		return errors.New(fmt.Sprintf("pushImage image:%s fail, open logFile fail.%s", imageId, err) )
	}
	
	defer logWrite.Close()
	
	opts := docker.PushImageOptions{
		Name:		imageId,
		Tag:		"latest",
		Registry:	j.conf.Builder.DockerRegistry,
		OutputStream: logWrite,
	}
	err = dockerClient.PushImage(opts, docker.AuthConfiguration{})
	
	if err != nil {
		return errors.New(fmt.Sprintf("pushImage image:%s fail.%s", imageId, err) )
	}
	j.logger.Infof("push image to private docker registry imageId:%s  success", imageId)
	end := time.Now()
	j.logger.Infof("pushImage image:%s, success timer:%s", imageId, end.Sub(start).Seconds() )
	
	return nil
}

//修改数据库状态
func (j *Job) updateDb(task *Building, logurl, msg string) {
	
	loginfo,err := os.Stat(logurl)
	cloudLogUrl := ""
	if err != nil {
		j.logger.Warnf("updateDb parse build log url fail.%s", err)
	}else{
		cloudLogUrl = fmt.Sprintf("%s/%s/%s", j.conf.Jss.Host, j.conf.Jss.BuildLogBucket, loginfo.Name())
	}
	
	
	setup := fmt.Sprintf("setup guid:%s,setupId:%s", task.AppGuid, task.StepsId)
	url := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", j.conf.Db.User, j.conf.Db.Passwd, j.conf.Db.Host, j.conf.Db.Port, j.conf.Db.Database)
	j.logger.Infof("begin update %s status to db ,url:%s", setup, url)
	
	db ,err := sql.Open("mysql", url)
	if err != nil {
		j.logger.Errorf("open sql:%s, fail.%s", url, err)
		return
	}
	defer db.Close()
	stmt, err := db.Prepare("update task_setup set start_time=?,end_time=?, status=?, log_url=?, msg=? where id=?")
	if err != nil {
		j.logger.Errorf("create %s, statement fail. %s",  setup, err)
		return
	}
	defer stmt.Close()
	_, err = stmt.Exec(task.StartTime, task.EndTime, task.Status, cloudLogUrl, msg, task.StepsId)
	
	if err != nil {
		j.logger.Errorf("update %s status to db fail. %s",  setup, err)
		return
	}
	j.logger.Infof("update %s status to db success", setup)
}

