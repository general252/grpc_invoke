package server

import (
	"context"
	"encoding/json"
	"github.com/general252/grpc_invoke/pkg/stub"
	"github.com/general252/grpc_invoke/static"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

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

func (tis *HttpServer) AddService(name string, host string, port int) {
	tis.clientsMux.Lock()
	defer tis.clientsMux.Unlock()

	for _, cli := range tis.clients {
		if cli.Host() == host && cli.Port() == port {
			return
		}
	}

	cli := stub.NewStub(host, port)
	err := cli.Connect(context.TODO())
	log.Printf("connect [%v] [%v:%v] %v", name, host, port, err)

	tis.clients = append(tis.clients, cli)
}

func (tis *HttpServer) router() {
	api := tis.r.Group("/rpc")

	tis.r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/rpc/ui")
	})

	api.StaticFS("/ui", static.GetFileSystem())                                 // 静态文件
	api.GET("/services", tis.routerServices)                                    // 获取service列表
	api.GET("/jsonSchema/:ServiceName/:MethodName", tis.routerMethodJsonSchema) // 获取method的Schema
	api.POST("/invoke/:ServiceName/:MethodName", tis.routerInvoke)              // 调用method
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
			schema := objectMethod.GetRequestJsonSchema()

			c.JSON(http.StatusOK, schema)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{})
}

func (tis *HttpServer) routerInvoke(c *gin.Context) {
	serviceName := c.Param("ServiceName")
	methodName := c.Param("MethodName")

	clients := tis.clients

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{})
		return
	}

	log.Println(string(body))

	for _, cli := range clients {
		if _, ok := cli.GetServerInfo().GetMethod(serviceName, methodName); ok {
			if resp, err := cli.InvokeRPC(c.Request.Context(), serviceName, methodName, string(body)); err != nil {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			} else {
				var obj map[string]any
				_ = json.Unmarshal([]byte(resp), &obj)
				c.JSON(http.StatusOK, obj)
			}

			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{})
}
