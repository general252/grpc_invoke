package stub

import (
	"bytes"
	"context"
	"fmt"
	"google.golang.org/grpc/connectivity"
	"log"

	"github.com/golang/protobuf/jsonpb"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

type Stub struct {
	host string
	port int

	conn *grpc.ClientConn
	cli  *grpcreflect.Client

	msgFactory *dynamic.MessageFactory

	serviceSymbols map[string]*ObjectFileDescriptor
	server         *JsonServer
}

func NewStub(host string, port int) *Stub {
	return &Stub{
		host:           host,
		port:           port,
		serviceSymbols: map[string]*ObjectFileDescriptor{},
		server:         &JsonServer{},
	}
}

func (tis *Stub) GetState() connectivity.State {
	if tis.conn == nil {
		return connectivity.Shutdown
	}

	return tis.conn.GetState()
}

func (tis *Stub) Host() string {
	return tis.host
}
func (tis *Stub) Port() int {
	return tis.port
}

func (tis *Stub) Connect(ctx context.Context) error {
	target := fmt.Sprintf("%v:%v", tis.host, tis.port)

	conn, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Println(err)
		return err
	}

	conn.GetState()
	tis.conn = conn
	tis.cli = grpcreflect.NewClientV1Alpha(context.TODO(), grpc_reflection_v1alpha.NewServerReflectionClient(tis.conn))

	if err = tis.loadServiceInfo(); err != nil {
		log.Println(err)
		return err
	}

	{
		for _, descriptor := range tis.serviceSymbols {
			for _, serviceDescriptor := range descriptor.GetFileDescriptor().GetServices() {
				objectService := &JsonService{
					Name:    serviceDescriptor.GetFullyQualifiedName(),
					Methods: []*JsonMethod{},
				}

				for _, methodDescriptor := range serviceDescriptor.GetMethods() {
					if methodDescriptor.IsServerStreaming() || methodDescriptor.IsClientStreaming() {
						log.Printf("[stream] %v, server stream: %v, client stream: %v",
							methodDescriptor.GetFullyQualifiedName(), methodDescriptor.IsServerStreaming(), methodDescriptor.IsClientStreaming())
						continue
					}

					objectMethod := &JsonMethod{
						Name:     methodDescriptor.GetName(),
						Request:  methodDescriptor.GetInputType().GetName(),
						Response: methodDescriptor.GetOutputType().GetName(),
						mtd:      methodDescriptor,
					}
					objectService.Methods = append(objectService.Methods, objectMethod)
				}

				tis.server.Services = append(tis.server.Services, objectService)
			}
		}
	}

	var ext dynamic.ExtensionRegistry
	tis.msgFactory = dynamic.NewMessageFactoryWithExtensionRegistry(&ext)

	return nil
}

// InvokeRPC grpc调用
// requestJsonData: proto.Message json
// return: proto.Message json
func (tis *Stub) InvokeRPC(ctx context.Context, service, method string, requestJsonData string) (string, error) {
	objectMethod, ok := tis.server.GetMethod(service, method)
	if !ok {
		return "", fmt.Errorf("not found [%v:%v]", service, method)
	}
	mtd := objectMethod.GetMethodDescriptor()

	var req = tis.msgFactory.NewMessage(mtd.GetInputType())
	if err := jsonpb.Unmarshal(bytes.NewBufferString(requestJsonData), req); err != nil {
		return "", err
	}

	stub := grpcdynamic.NewStubWithMessageFactory(tis.conn, tis.msgFactory)
	resp, err := stub.InvokeRpc(ctx, mtd, req)
	if err != nil {
		return "", err
	} else if false {
		dm := resp.(*dynamic.Message)
		fd := dm.GetMessageDescriptor().FindFieldByName("message")
		_ = dm.GetField(fd)
	}

	respStr, err := new(jsonpb.Marshaler).MarshalToString(resp)
	if err != nil {
		return "", err
	}

	return respStr, nil
}

func (tis *Stub) GetObjectFileSymbol() map[string]*ObjectFileDescriptor {
	return tis.serviceSymbols
}

func (tis *Stub) GetServerInfo() *JsonServer {
	return tis.server
}

func (tis *Stub) loadServiceInfo() error {
	cli := tis.cli
	serviceSymbols, err := cli.ListServices()
	if err != nil {
		log.Println(err)
		return err
	}

	for _, symbolName := range serviceSymbols {
		if symbolName == "grpc.reflection.v1alpha.ServerReflection" {
			continue
		}

		fileDesc, err := cli.FileContainingSymbol(symbolName)
		if err != nil {
			log.Println(err)
			return err
		}

		tis.serviceSymbols[symbolName] = &ObjectFileDescriptor{
			symbolName: symbolName,
			fileDesc:   fileDesc,
		}
	}

	return nil
}

type ObjectFileDescriptor struct {
	symbolName string
	fileDesc   *desc.FileDescriptor
}

func (tis *ObjectFileDescriptor) GetSymbolName() string {
	return tis.symbolName
}

func (tis *ObjectFileDescriptor) GetFileDescriptor() *desc.FileDescriptor {
	return tis.fileDesc
}
