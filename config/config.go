package config

//配置文件
import (
	"github.com/cloudfoundry-incubator/candiedyaml"
	"io/ioutil"
)

// 日志配置
type LoggingConfig struct {
	File  string `yaml:"file"`
	Level string `yaml:"level"`
}

// 日志配置默认值
var defaultLoggingConfig = LoggingConfig{
	File: "/export/home/jae/builder.log",
	Level: "debug",
}

// builder config
type BuilderConfig struct {
	Port					string 				`yaml:"port"`
	HandlerLogging			bool				`yaml:"handlerLogging"`
	DockerFilePath			string				`yaml:"dockerfile_path"`
	DockerFileRemotePath	string				`yaml:"dockerfile_remote_path"`
	BuildPath				string				`yaml:"build_path"`
	SnapshotPath			string				`yaml:"snapshot_path"`
	WorkThreadSize			int					`yaml:"work_thread_size"`	
	Languages				[]string			`yaml:"build_languages"`
	DockerUrl				string				`yaml:"docker_remoute_url"`
	DockerRegistry			string				`yaml:"docker_registry"`
	JobIntervalSecond		int					`yaml:"job_interval_second"`
}

var defaultBuilder = BuilderConfig{
	Port:					"8081",
	HandlerLogging:			true,
	BuildPath:				"/export/home/jae/builder",
	DockerFilePath:			"/export/home/builder/dockerfile/",
	DockerFileRemotePath:	"http://127.0.0.1/packages/dockerfile/",
	SnapshotPath:			"/export/home/builder/",
	WorkThreadSize:			1,
	Languages:				[]string{"JAVA","PHP","PYTHON","RUBY","NODEJS"},
	DockerUrl:				"http://127.0.0.1:4243",
	DockerRegistry:			"docker.registry.com",
	JobIntervalSecond:		5,
}

//db config
type DbConfig	struct {
	Host			string				`yaml:"host"`
	Port			string				`yaml:"port"`
	User			string				`yaml:"user"`
	Passwd			string				`yaml:"pwd"`	
	Database		string				`yaml:"database"`
}

var defaultDb = DbConfig{
	Host:			"192.168.192.131",
	Port:			"3306",
	User:			"root",
	Passwd:			"root",	
	Database:		"jae",
}

type JssConfig  struct {
	AccessKey			string			`yaml:"access_key"`
	SecretKey			string			`yaml:"secret_Key"`
	BuildLogBucket		string			`yaml:"buildlog_bucket"`
	AppPackageBucket	string			`yaml:"app_package_bucket"`
	Host				string			`yaml:"host"`
	Domain				string			`yaml:"domain"`
	TimeOut				int				`yaml:"timeout_second"`
}

var defaultJssConfig = JssConfig{
	AccessKey:		    "3e05c77d08a14045a0bd2ea307eb1ae9",
	SecretKey:		    "6212b165e5b54c3cb43d20295bf03e7aPIymHVqg",
	BuildLogBucket:		"jae-docker-build-log",
	AppPackageBucket:	"jae-apppackage",
	Host:				"storage.jcloud.com",
	Domain:				"http://storage.jcloud.com",
	TimeOut:			60,
}

type Config struct {
	Logging			 LoggingConfig              		`yaml:"logging"`
	Builder			 BuilderConfig						`yaml:"builder"`
	Db				 DbConfig							`yaml:"db"`
	Jss			     JssConfig							`yaml:"jss"`
}

var defaultConfig	 = Config {
	Logging:	defaultLoggingConfig,
	Builder:	defaultBuilder,
	Db:			defaultDb,
	Jss:		defaultJssConfig,
}

func DefaultConfig() *Config{
	
	c := defaultConfig
	
	return &c
}

//解析配置文件,初始化配置对象
func (c *Config) Initialize(configYAML [] byte) error{

	return candiedyaml.Unmarshal(configYAML, &c)
}

//根据文件初始化配置对象
func InitConfigFromFile(path string) *Config{

	var c *Config = DefaultConfig()
	var e error
	
	b, e := ioutil.ReadFile(path)
	
	if e != nil {
		panic(e.Error())
	}
	
	e = c.Initialize(b)
	
	return c
}