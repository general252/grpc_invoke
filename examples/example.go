package examples

import (
	"context"
	"log"
	"net"

	"github.com/general252/grpc_invoke/examples/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type HelloService struct {
	helloworld.UnimplementedGreeterServer
}

func (c *HelloService) SayHello(_ context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	resp := &helloworld.HelloReply{
		Message: "hello " + req.GetName(),
	}
	return resp, nil
}

func (c *HelloService) GetVersion(context.Context, *helloworld.GetVersionReq) (*helloworld.GetVersionReply, error) {
	return &helloworld.GetVersionReply{Version: "1.0.2"}, nil
}

func (c *HelloService) ClientStream(stream helloworld.Greeter_ClientStreamServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}

		log.Println(msg.GetData())
	}
}

func RunHelloServer() (port int, err error) {
	service := &HelloService{}
	rpcServer := grpc.NewServer()
	helloworld.RegisterGreeterServer(rpcServer, service)
	reflection.Register(rpcServer)

	lis, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: 0})
	if err != nil {
		log.Println(err)
		return 0, err
	}

	go func() {
		defer lis.Close()
		_ = rpcServer.Serve(lis)
	}()

	return lis.Addr().(*net.TCPAddr).Port, nil
}
