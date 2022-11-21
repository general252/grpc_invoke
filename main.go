package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/general252/grpc_invoke/examples"
	"github.com/general252/grpc_invoke/pkg/config"
	"github.com/general252/grpc_invoke/pkg/server"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	serv := server.NewHttpServer()
	go serv.Server(8888)
	defer serv.Close()

	if address := traefilServices(); address != nil {
		for _, addr := range address {
			serv.AddService(addr.Name, addr.Host, addr.Port)
		}
	}

	if port, err := examples.RunHelloServer(); err == nil {
		log.Printf("测试gRPC服务端口: %v", port)
		// serv.AddService("example", "127.0.0.1", port)
	}

	cfg := config.GetConfig()
	for _, service := range cfg.Services {
		serv.AddService(service.Name, service.Host, service.Port)
	}

	quitChan := make(chan os.Signal, 2)
	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	<-quitChan
}

func traefilServices() []config.Service {
	resp, err := http.Get("http://127.0.0.1:58181/api/http/services?search=&status=&per_page=120&page=1")
	if err != nil {
		log.Println(err)
		return nil
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil
	}

	type JsonLoadBalancer struct {
		Status       string            `json:"status"`       // enabled
		ServerStatus map[string]string `json:"serverStatus"` // "h2c://127.0.0.1:60038": "UP"
		Name         string            `json:"name"`
		Provider     string            `json:"provider"` // rest
		Type         string            `json:"type"`     // loadbalancer, weighted
	}

	var objets []JsonLoadBalancer
	if err = json.Unmarshal(data, &objets); err != nil {
		log.Println(err)
		return nil
	}

	var address []config.Service

	for _, object := range objets {
		if object.Type != "loadbalancer" {
			continue
		}

		if object.ServerStatus == nil {
			continue
		}

		re, err := regexp.Compile("[0-9.]+[0-9.]+")
		if err != nil {
			panic(err)
		}

		for k, v := range object.ServerStatus {
			if v != "UP" || !strings.HasPrefix(k, "h2c://") {
				continue
			}

			addr := re.FindAllString(k, -1)
			if len(addr) != 2 {
				continue
			}

			host := addr[0]
			port, err := strconv.ParseInt(addr[1], 10, 64)
			if err != nil {
				continue
			}

			address = append(address, config.Service{
				Name: object.Name,
				Host: host,
				Port: int(port),
			})

		}
	}

	return address
}
