// Copyright 2014 JD Inc. All Rights Reserved.
// Author: zhangwei
// email : zhangwei_2943@163.com
// date  : 2014-12-19
//==============================================================
// JAE打包系统,负责根据用户的代码信息,使用环境信息动态创建Dockerfile
// 文件,然后build成image,最后将上传到private registry
//==============================================================
package main 

import (
	steno "github.com/cloudfoundry/gosteno"
	"builder/config"
	codec "builder/logging"
	"builder/controller"
	"builder/building"
	"flag"
	"os"
	"fmt"
)

var (
		conf		 		*config.Config
		logger     			*steno.Logger
		configFile 			string
		ctrl  				*controller.Controller
		handler				*building.BuildingHandler
	)

//初始化配置信息
func setupConfig () {

	fmt.Printf("setup config filePath:%s \n",configFile)
	
	conf = config.DefaultConfig()
	
	if configFile != "" {
		conf = config.InitConfigFromFile(configFile);
	}
}

//初始化日志信息
func setupLogger () {

	fmt.Printf("setup logger level:%s,file:%s \n",conf.Logging.Level,conf.Logging.File)
	
	l, err := steno.GetLogLevel(conf.Logging.Level)
	if err != nil {
		logger.Errorf("steno.GetLogLevel fail , %s", err)
		os.Exit(1)
	}

	s := make([]steno.Sink, 0, 3)
	s = append(s, steno.NewFileSink(conf.Logging.File))

	stenoConfig := &steno.Config{
		Sinks: s,
		Codec: codec.NewStringCodec(),
		Level: l,
	}

	steno.Init(stenoConfig)
	
	logger = steno.NewLogger("builder")
}

func setupBuildHandler () {
	handler = building.NewBuilding(conf)	
}

func setupController() {
	ctrl = controller.NewController(conf, handler)	
}


//解析启动参数,-c 配置文件路径
func init(){
	flag.StringVar(&configFile, "c", "", "Configuration File")
	
	flag.Parse()
}

func setup () {
	setupConfig()
	setupLogger()
	setupBuildHandler()
	setupController()
}

func start () {
	handler.Start()
	ctrl.ServeApi()
}

func main() {
	setup()
	start()
}

