package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/general252/grpc_invoke/pkg/config"
	"github.com/general252/grpc_invoke/pkg/http_swagger"
	"github.com/general252/grpc_invoke/pkg/stub"
	"github.com/general252/grpc_invoke/static"
	"google.golang.org/grpc/metadata"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type HttpServer struct {
	lis *net.TCPListener
	r   *gin.Engine

	clients    []*stub.Stub
	clientsMux sync.Mutex

	apis []*http_swagger.JsonAPI
}

func NewHttpServer() *HttpServer {
	return &HttpServer{
		clients: []*stub.Stub{},
	}
}

func (tis *HttpServer) Server(port int) error {
	tis.r = gin.Default()
	tis.router()

	l, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: port})
	if err != nil {
		return err
	}

	tis.lis = l

	tis.loadSwaggerFile()

	if err = tis.r.RunListener(l); err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (tis *HttpServer) Port() int {
	if tis.lis == nil {
		return 0
	}

	if addr, ok := tis.lis.Addr().(*net.TCPAddr); ok {
		return addr.Port
	}

	return 0
}

func (tis *HttpServer) Close() {
	_ = tis.lis.Close()
}

func (tis *HttpServer) AddService(name string, host string, port int) error {
	tis.clientsMux.Lock()
	defer tis.clientsMux.Unlock()

	for _, cli := range tis.clients {
		if cli.Host() == host && cli.Port() == port {
			return fmt.Errorf("already exists")
		}
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()

	cli := stub.NewStub(host, port)
	if err := cli.Connect(ctx); err != nil {
		log.Printf("connect [%v] [%v:%v] %v", name, host, port, err)
		return err
	}

	tis.clients = append(tis.clients, cli)
	return nil
}

func (tis *HttpServer) router() {
	api := tis.r.Group("/rpc")

	tis.r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/rpc/ui")
	})

	api.StaticFS("/ui", static.GetFileSystem()) // 静态文件
	api.POST("/services", tis.routerAddService)
	api.GET("/services", tis.routerServices)                                    // 获取service列表
	api.GET("/jsonSchema/:ServiceName/:MethodName", tis.routerMethodJsonSchema) // 获取method的Schema
	api.POST("/invoke/:ServiceName/:MethodName", tis.routerInvoke)              // 调用method

	swaggerApi := tis.r.Group("/swagger")
	swaggerApi.GET("/services", tis.swServices)
	swaggerApi.GET("/jsonSchema/:ServiceName/:MethodName", tis.swServicesJsonSchema)
	swaggerApi.POST("/invoke/:ServiceName/:MethodName", tis.swRouterInvoke)
}

type JsonAddServiceRequest struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (tis *HttpServer) routerAddService(c *gin.Context) {
	var request JsonAddServiceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := tis.AddService(request.Name, request.Host, request.Port); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{})
}

func (tis *HttpServer) routerServices(c *gin.Context) {
	clients := tis.clients

	var response []*stub.JsonService
	for _, cli := range clients {
		response = append(response, cli.GetServerInfo().Services...)
	}

	c.JSON(http.StatusOK, response)
}

func (tis *HttpServer) routerMethodJsonSchema(c *gin.Context) {
	serviceName := c.Param("ServiceName")
	methodName := c.Param("MethodName")

	clients := tis.clients

	log.Println(serviceName)
	log.Println(methodName)

	for _, cli := range clients {
		if objectMethod, ok := cli.GetServerInfo().GetMethod(serviceName, methodName); ok {
			inSchema := objectMethod.GetRequestJsonSchema()
			outSchema := objectMethod.GetResponseJsonSchema()

			c.JSON(http.StatusOK, gin.H{
				"input":  inSchema,
				"output": outSchema,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": "not found",
	})
}

type JsonInvokeRequest struct {
	Header map[string]string `json:"header"`
	Data   json.RawMessage   `json:"data"`
}

type JsonInvokeReply struct {
	Header  metadata.MD    `json:"header"`
	Trailer metadata.MD    `json:"trailer"`
	Data    map[string]any `json:"data"`
}

func (tis *HttpServer) routerInvoke(c *gin.Context) {
	serviceName := c.Param("ServiceName")
	methodName := c.Param("MethodName")

	clients := tis.clients

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	var objectRequest JsonInvokeRequest
	if err = json.Unmarshal(body, &objectRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if body, err = objectRequest.Data.MarshalJSON(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	for _, cli := range clients {
		if _, ok := cli.GetServerInfo().GetMethod(serviceName, methodName); ok {
			// 执行
			if resp, header, trailer, err := cli.InvokeRPC(c.Request.Context(), serviceName, methodName, string(body), objectRequest.Header); err != nil {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			} else {
				// 回复
				var object map[string]any
				_ = json.Unmarshal([]byte(resp), &object)
				c.JSON(http.StatusOK, &JsonInvokeReply{
					Header:  header,
					Trailer: trailer,
					Data:    object,
				})
			}

			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{})
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type JsonSwaggerService struct {
	Name    string                      `json:"service_name"`
	Methods []*JsonSwaggerServiceMethod `json:"methods"`
}

type JsonSwaggerServiceMethod struct {
	Path   string `json:"path"`   //
	Method string `json:"method"` // get, post, delete, put
}

func (tis *HttpServer) swServices(c *gin.Context) {
	var response []*JsonSwaggerService
	for _, api := range tis.apis {
		item := &JsonSwaggerService{
			Name:    api.String(),
			Methods: []*JsonSwaggerServiceMethod{},
		}

		for _, method := range api.Methods {
			item.Methods = append(item.Methods, &JsonSwaggerServiceMethod{
				Path:   method.Path,
				Method: method.Method,
			})
		}

		response = append(response, item)
	}

	c.JSON(http.StatusOK, response)
}

func (tis *HttpServer) swServicesJsonSchema(c *gin.Context) {
	serviceName := c.Param("ServiceName")
	methodName := c.Param("MethodName")

	for _, api := range tis.apis {
		if api.String() == serviceName {
			for _, method := range api.Methods {
				if method.Path == methodName {
					var objectInput map[string]any
					_ = json.Unmarshal([]byte(method.Input), &objectInput)

					var objectOutput map[string]any
					_ = json.Unmarshal([]byte(method.Output), &objectOutput)

					c.JSON(http.StatusOK, gin.H{
						"input":  objectInput,
						"output": objectOutput,
					})
				}
			}
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": "not found",
	})
}

// swRouterInvoke 废除, 使用代理的方式解决http调用
func (tis *HttpServer) swRouterInvoke(c *gin.Context) {
	serviceName := c.Param("ServiceName")
	methodName := c.Param("MethodName")

	var objectMethod *http_swagger.JsonMethod
	var objectAPI *http_swagger.JsonAPI
	for _, api := range tis.apis {
		if api.String() == serviceName {
			for _, method := range api.Methods {
				if method.Path == methodName {
					objectMethod = &method
					objectAPI = api
				}
			}
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second*30)
	defer cancel()

	uri := fmt.Sprintf("%v/%v", objectAPI.String(), objectMethod.Path)
	req, err := http.NewRequestWithContext(ctx, objectMethod.Method, uri, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("new request fail. %v", err),
		})
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("get response fail. %v", err),
		})
		return
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("read response fail. %v", err),
		})
		return
	}

	var objectOutput map[string]any
	_ = json.Unmarshal(data, &objectOutput)

	c.JSON(http.StatusOK, data)
}

func (tis *HttpServer) loadSwaggerFile() {
	dir, _ := config.GetExeDir()
	swaggerDir := fmt.Sprintf("%v/swagger", dir)

	var swaggerFiles []string
	_ = filepath.Walk(swaggerDir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		swaggerFiles = append(swaggerFiles, path)

		return nil
	})

	log.Println(swaggerFiles)

	for _, swaggerFile := range swaggerFiles {
		data, err := os.ReadFile(swaggerFile)
		if err != nil {
			continue
		}
		api, err := http_swagger.ParseSwagger(data)
		if err != nil {
			continue
		}

		tis.apis = append(tis.apis, api)
	}
}
