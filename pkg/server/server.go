package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/general252/grpc_invoke/pkg/stub"
	"github.com/general252/grpc_invoke/static"
	"google.golang.org/grpc/metadata"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type HttpServer struct {
	lis *net.TCPListener
	r   *gin.Engine

	clients    []*stub.Stub
	clientsMux sync.Mutex
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
