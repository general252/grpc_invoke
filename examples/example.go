package examples

import (
	"context"
	"encoding/json"
	"google.golang.org/grpc/metadata"
	"log"
	"net"

	"github.com/general252/grpc_invoke/examples/helloworld"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type HelloService struct {
	helloworld.UnimplementedGreeterServer
}

func (c *HelloService) SayHello(ctx context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	if md, ok := metadata.FromIncomingContext(ctx); !ok {
		log.Printf("get metadata error")
	} else {
		headerStr, _ := json.MarshalIndent(md, "", "  ")
		log.Printf(">>>> header: %v", string(headerStr))

		// create and send header
		header := metadata.Pairs("header-key", "val")
		_ = grpc.SendHeader(ctx, header)

		// create and set trailer
		trailer := metadata.Pairs("trailer-key", "val")
		_ = grpc.SetTrailer(ctx, trailer)
	}

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
