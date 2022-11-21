package stub

import (
	"github.com/general252/grpc_invoke/pkg/schema"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"strings"
)

type JsonServer struct {
	Services []*JsonService `json:"services"`
}

func (tis *JsonServer) GetMethod(serviceName, methodName string) (*JsonMethod, bool) {
	for _, service := range tis.Services {
		if service.Name == serviceName {
			for _, method := range service.Methods {
				if method.Name == methodName {
					return method, true
				}
			}
		}
	}

	return nil, false
}

type JsonService struct {
	Name    string        `json:"service_name"`
	Methods []*JsonMethod `json:"methods"`
}

type JsonMethod struct {
	Name     string `json:"method_name"`
	Request  string `json:"request"`
	Response string `json:"response"`
	mtd      *desc.MethodDescriptor
}

func (tis *JsonMethod) GetMethodDescriptor() *desc.MethodDescriptor {
	return tis.mtd
}

func (tis *JsonMethod) GetRequestJsonSchema() *schema.JsonSchema {
	inputType := tis.mtd.GetInputType()

	result := MessageToSchema(inputType, true)

	return result
}

func MessageToSchema(msg *desc.MessageDescriptor, root bool) *schema.JsonSchema {

	var result = &schema.JsonSchema{
		Title:       msg.GetName(),
		Type:        "object",
		Description: msg.GetFullyQualifiedName(),
		Properties:  map[string]*schema.JsonSchema{},
		Options: map[string]any{
			"collapsed": !root,
		},
	}

	var getDescriptor = func(field *desc.FieldDescriptor) string {
		// return fmt.Sprintf("%v(%v)", field.GetFullyQualifiedName(), field.GetType().String())
		v := field.GetType().String()
		if strings.HasPrefix(v, "TYPE_") {
			v = v[len("TYPE_"):]
		}

		v = strings.ToLower(v)
		return v
	}

	for _, fieldDescriptor := range msg.GetFields() {
		one := &schema.JsonSchema{
			Title:       fieldDescriptor.GetName(),
			Type:        "",
			Description: getDescriptor(fieldDescriptor),
			Default:     nil,
			Enum:        nil,
			UniqueItems: false,
			Items:       nil,
			MinLength:   0,
			MiniNum:     0,
			MaxiNum:     0,
			Format:      "",
			Properties:  nil,
		}

		switch fieldDescriptor.GetType() {
		case descriptor.FieldDescriptorProto_TYPE_BOOL:
			one.Type = schema.JsonSchemaTypeBoolean
		case descriptor.FieldDescriptorProto_TYPE_DOUBLE,
			descriptor.FieldDescriptorProto_TYPE_FLOAT:
			one.Type = schema.JsonSchemaTypeNumber
		case descriptor.FieldDescriptorProto_TYPE_STRING,
			descriptor.FieldDescriptorProto_TYPE_BYTES:
			one.Type = schema.JsonSchemaTypeString
		case descriptor.FieldDescriptorProto_TYPE_INT64,
			descriptor.FieldDescriptorProto_TYPE_UINT64,
			descriptor.FieldDescriptorProto_TYPE_INT32,
			descriptor.FieldDescriptorProto_TYPE_FIXED64,
			descriptor.FieldDescriptorProto_TYPE_FIXED32,
			descriptor.FieldDescriptorProto_TYPE_UINT32,
			descriptor.FieldDescriptorProto_TYPE_SFIXED32,
			descriptor.FieldDescriptorProto_TYPE_SFIXED64,
			descriptor.FieldDescriptorProto_TYPE_SINT32,
			descriptor.FieldDescriptorProto_TYPE_SINT64:
			one.Type = schema.JsonSchemaTypeInteger
		case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
			one = MessageToSchema(fieldDescriptor.GetMessageType(), false)
			one.Type = schema.JsonSchemaTypeObject
			one.Description = getDescriptor(fieldDescriptor)
		case descriptor.FieldDescriptorProto_TYPE_ENUM:
			one.Type = schema.JsonSchemaTypeString
			for _, valueDescriptor := range fieldDescriptor.GetEnumType().GetValues() {
				one.Enum = append(one.Enum, valueDescriptor.GetName())
			}
		case descriptor.FieldDescriptorProto_TYPE_GROUP:

		}

		if fieldDescriptor.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED {
			one = &schema.JsonSchema{
				Title:       fieldDescriptor.GetName(),
				Type:        schema.JsonSchemaTypeArray,
				Description: getDescriptor(fieldDescriptor),
				Items:       one,
			}
		}

		result.Properties[fieldDescriptor.GetJSONName()] = one
	}

	return result
}
