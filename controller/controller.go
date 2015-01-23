package controller

import (
	"net/http"
 	"github.com/gorilla/mux"
	"builder/config"
	"builder/building"
	steno "github.com/cloudfoundry/gosteno"
	"net"
	"os"
)

// 提供restfull 接口
type Controller struct {
	conf 				*config.Config 
	logger     			*steno.Logger
	handler             *building.BuildingHandler
}

type HttpApiFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string) error

func NewController (c *config.Config, handler *building.BuildingHandler) *Controller {
	return &Controller {
		conf:				c,
		logger: 			steno.NewLogger("builder"),
		handler:			handler,
	}
}

func (c *Controller) returnJson(v interface{}, w http.ResponseWriter) error{
	data, err := encodeJson(v)
	
	if err != nil {
		c.logger.Errorf("encodejson fail,err:%v",err)
		return err
	}else {
		writeJson(data, w)
	}
	return nil
}

func (c *Controller) configInfo(w http.ResponseWriter, r *http.Request, vars map[string]string ) error {
	err := c.returnJson(c.conf, w)
	return err
}

func (c *Controller) builder(w http.ResponseWriter, r *http.Request, vars map[string]string ) error {
	
	formdate, err := Decode(r.Body)
	if err != nil {
		return err	
	}
	
	appguid 			:= vars["appguid"]
	setpId  			:= vars["setpid"]
	appname				:= formdate.getParam("app_name")
	language 			:= formdate.getParam("language")
	applicationServer 	:= formdate.getParam("application_server")
	runtimeEnvironment 	:= formdate.getParam("runtime_environment")
	
	task := &building.Building{
			AppGuid:			appguid,
			StepsId:			setpId,
			Language:			language,
			AppName:			appname,
			ApplicationServer:	applicationServer,
			RuntimeEnvironment:	runtimeEnvironment,
		}
	err = c.handler.Register(task)
	
	if err != nil {
		return err	
	}
	
	result := make(map[string]string)
	result["status"] = "200"
	err = c.returnJson(result, w)
	return err
}

//获取所有正在打包或等待打包的数据
func (c *Controller) getBuildings(w http.ResponseWriter, r *http.Request, vars map[string]string ) error {
	err := c.returnJson(c.handler.GetAllBuildings(), w)
	return err
}

func (c *Controller) searchBuildings(w http.ResponseWriter, r *http.Request, vars map[string]string ) error {
	appguid := vars["guid"]
	err := c.returnJson(c.handler.SearchBUildings(appguid), w)
	return err
}

func (c *Controller) makeHttpHandler(logging bool, localMethod string, localRouter string, handlerFunc HttpApiFunc) http.HandlerFunc {
	
	return func(w http.ResponseWriter, r *http.Request) {
		c.logger.Infof("Calling %s %s", localMethod, localRouter)
		
		if logging {
			c.logger.Infof("reqMethod:%s , reqURI:%s , userAgent:%s", r.Method, r.RequestURI, r.Header.Get("User-Agent"))
		}
		
		if err := handlerFunc(w , r, mux.Vars(r)) ; err != nil {
			c.logger.Errorf("Handler for %s %s returned error: %s", localMethod, localRouter, err)
			http.Error(w, err.Error(), 400)
		}
	}
}

func (c *Controller) createoRuter () (*mux.Router, error) {
	r := mux.NewRouter()
	
	m := map[string]map[string] HttpApiFunc {
		"GET": {
			"/configinfo":							c.configInfo,
			"/buildings":							c.getBuildings,
			"/buildings/{guid}/search":				c.searchBuildings,				
		},
		"POST": {
			"/{appguid}/{setpid}/build":			c.builder,
		},
		"DELETE": {
		
		},
		"PUT": {
		
		},
	}
	
	//遍历定义的方法,注册服务
	for method, routers := range m {
		
		for route, fct := range routers {
			c.logger.Infof("registering method:%s, router:%s", method, route)
			
			localRoute := route
			localFct   := fct
			localMethod := method
			
			//build the handler function
			f := c.makeHttpHandler(c.conf.Builder.HandlerLogging, localMethod, localRoute, localFct)
			
			if localRoute == "" {
				r.Methods(localMethod).HandlerFunc(f)
			}else {
				r.Path("/" + "builder"+ localRoute).Methods(localMethod).HandlerFunc(f)
			}
		}
	}
	
	return r, nil
}

// 开启服务监听
func (c *Controller) listenAndServe() error {
	
	var l net.Listener
	r, err := c.createoRuter()
	
	if err != nil {
		return err
	}
	
	addr := ":"+c.conf.Builder.Port
	
	l, err  = net.Listen("tcp", addr)
	if err != nil {
		c.logger.Errorf("listenAndServe fail, %s", err)	
		return err
	}
	httpSrv := http.Server{Addr: addr, Handler: r}
	
	return httpSrv.Serve(l)
}

func (c *Controller) ServeApi() {
	c.logger.Infof("starting service ........")
	err :=  c.listenAndServe()
	if err != nil {
		c.logger.Errorf("ServeApi error , %s", err)
		os.Exit(1)
	}
	
	c.logger.Infof("starting service success, port: %s", c.conf.Builder.Port)
}