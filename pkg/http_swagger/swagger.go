package http_swagger

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"log"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
)

const (
	Definitions = "#/definitions/"        // v2
	Components  = "#/components/schemas/" // v3
)

type JsonAPI struct {
	Scheme   string       `json:"scheme"`   // http
	Host     string       `json:"host"`     // 127.0.0.1
	Port     int          `json:"port"`     // 8080
	BashPath string       `json:"bashPath"` // /bvcr/v1
	Methods  []JsonMethod `json:"methods"`
}

func (tis *JsonAPI) String() string {
	return fmt.Sprintf("%v://%v:%v%v", tis.Scheme, tis.Host, tis.Port, tis.BashPath)
}

type JsonMethod struct {
	Path   string `json:"path"`   // /crearo/filter
	Method string `json:"method"` // get, post, delete, put
	Input  string `json:"input"`  // json schema
	Output string `json:"output"` // json schema
}

// ParseSwagger 解析swagger.json文件
func ParseSwagger(data []byte) (*JsonAPI, error) {
	v, err := parseSwaggerV2(data)
	if err == nil {
		return v, nil
	}

	return parseSwaggerV3(data)
}

func extendSchemaRefV2(schema *openapi3.SchemaRef, s *openapi2.T) *openapi3.SchemaRef {
	if schema == nil || s == nil {
		return nil
	}
	if len(schema.Ref) != 0 {
		key := strings.TrimPrefix(schema.Ref, Definitions)
		if v, ok := s.Definitions[key]; ok {
			schema.Ref = ""
			schema.Value = v.Value
		}
	}

	extendSchemaRefV2(schema.Value.Items, s)

	for _, ref := range schema.Value.Properties {
		extendSchemaRefV2(ref, s)
	}

	return schema
}

func parseSwaggerV2(data []byte) (*JsonAPI, error) {
	s2 := openapi2.T{}
	if err := s2.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	s2.Host = strings.ReplaceAll(s2.Host, "localhost", "127.0.0.1")
	addr, err := netip.ParseAddrPort(s2.Host)
	if err != nil {
		return nil, err
	}

	var api = &JsonAPI{
		Scheme:   "http",
		Host:     addr.Addr().String(),
		Port:     int(addr.Port()),
		BashPath: s2.BasePath,
		Methods:  []JsonMethod{},
	}
	if len(s2.Schemes) > 0 {
		api.Scheme = s2.Schemes[0]
	}

	var updateMethod = func(obj *JsonMethod, src *openapi2.Operation) {

		for _, parameter := range src.Parameters {
			if parameter.Schema == nil {
				continue
			}

			parameter.Schema = extendSchemaRefV2(parameter.Schema, &s2)
			if data, err := parameter.Schema.MarshalJSON(); err == nil {
				obj.Input = string(data)
			}
		}

		for responseCode, response := range src.Responses {
			if response.Schema == nil {
				continue
			}
			if code, err := strconv.Atoi(responseCode); err != nil {
				log.Println(err)
				continue
			} else if code >= http.StatusOK && code < http.StatusMultipleChoices {
				// 成功, 继续
			} else {
				log.Println(code)
				continue
			}

			response.Schema = extendSchemaRefV2(response.Schema, &s2)
			if data, err := response.Schema.MarshalJSON(); err == nil {
				obj.Output = string(data)
			}
		}
	}

	for methodPath, item := range s2.Paths {
		var obj = JsonMethod{
			Path:   methodPath,
			Method: http.MethodGet,
			Input:  "",
			Output: "",
		}

		if item.Get != nil {
			obj.Method = http.MethodGet
			updateMethod(&obj, item.Get)
		} else if item.Post != nil {
			obj.Method = http.MethodPost
			updateMethod(&obj, item.Post)
		} else if item.Put != nil {
			obj.Method = http.MethodPut
			updateMethod(&obj, item.Put)
		} else if item.Delete != nil {
			obj.Method = http.MethodDelete
			updateMethod(&obj, item.Delete)
		}

		api.Methods = append(api.Methods, obj)
	}

	return api, nil
}

func extendSchemaRefV3(schema *openapi3.SchemaRef, s *openapi3.T) *openapi3.SchemaRef {
	if schema == nil || s == nil {
		return nil
	}
	if len(schema.Ref) != 0 {
		key := strings.TrimPrefix(schema.Ref, Components)
		if v, ok := s.Components.Schemas[key]; ok {
			schema.Ref = ""
			schema.Value = v.Value
		}
	}

	extendSchemaRefV3(schema.Value.Items, s)

	for _, ref := range schema.Value.Properties {
		extendSchemaRefV3(ref, s)
	}

	return schema
}

func parseSwaggerV3(data []byte) (*JsonAPI, error) {
	s2 := openapi3.T{}
	if err := s2.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	if len(s2.Servers) == 0 {
		return nil, fmt.Errorf("no Servers")
	}

	s2.Servers[0].URL = strings.ReplaceAll(s2.Servers[0].URL, "localhost", "127.0.0.1")
	urlInfo, err := url.Parse(s2.Servers[0].URL)
	if err != nil {
		return nil, err
	}

	addr, err := netip.ParseAddrPort(urlInfo.Host)
	if err != nil {
		return nil, err
	}

	var api = &JsonAPI{
		Scheme:   urlInfo.Scheme,
		Host:     addr.Addr().String(),
		Port:     int(addr.Port()),
		BashPath: urlInfo.Path,
		Methods:  []JsonMethod{},
	}

	var updateMethod = func(obj *JsonMethod, src *openapi3.Operation) {
		if src.RequestBody != nil && src.RequestBody.Value != nil {
			for _, mediaType := range src.RequestBody.Value.Content {
				if mediaType.Schema == nil {
					continue
				}

				mediaType.Schema = extendSchemaRefV3(mediaType.Schema, &s2)
				if data, err := mediaType.Schema.MarshalJSON(); err == nil {
					obj.Input = string(data)
				}
			}
		}

		for responseCode, response := range src.Responses {
			if response.Value == nil || response.Value.Content == nil {
				continue
			}
			if code, err := strconv.Atoi(responseCode); err != nil {
				log.Println(err)
				continue
			} else if code >= http.StatusOK && code < http.StatusMultipleChoices {
				// 成功, 继续
			} else {
				log.Println(code)
				continue
			}

			for _, mediaType := range response.Value.Content {
				if mediaType.Schema == nil {
					continue
				}

				mediaType.Schema = extendSchemaRefV3(mediaType.Schema, &s2)
				if data, err := mediaType.Schema.MarshalJSON(); err == nil {
					obj.Output = string(data)
				}
			}
		}
	}

	for methodPath, item := range s2.Paths {
		var obj = JsonMethod{
			Path:   methodPath,
			Method: http.MethodGet,
			Input:  "",
			Output: "",
		}

		if item.Get != nil {
			obj.Method = http.MethodGet
			updateMethod(&obj, item.Get)
		} else if item.Post != nil {
			obj.Method = http.MethodPost
			updateMethod(&obj, item.Post)
		} else if item.Put != nil {
			obj.Method = http.MethodPut
			updateMethod(&obj, item.Put)
		} else if item.Delete != nil {
			obj.Method = http.MethodDelete
			updateMethod(&obj, item.Delete)
		}

		api.Methods = append(api.Methods, obj)
	}

	return api, nil
}
