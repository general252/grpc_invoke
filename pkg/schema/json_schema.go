package schema

import (
	"encoding/json"
	"log"
)

type JsonSchema struct {
	Title       string         `json:"title,omitempty"`
	Type        JsonSchemaType `json:"type"` // object, integer, string, array, number, boolean, null
	Description string         `json:"description,omitempty"`

	Default any `json:"default,omitempty"`

	Enum []string `json:"enum,omitempty"`

	UniqueItems bool        `json:"uniqueItems,omitempty"` // items约束
	Items       *JsonSchema `json:"items,omitempty"`

	MinLength int    `json:"minLength,omitempty"`
	MiniNum   int    `json:"minimum,omitempty"`
	MaxiNum   int    `json:"maximum,omitempty"`
	Format    string `json:"format,omitempty"`

	Options map[string]any `json:"options"`

	Properties map[string]*JsonSchema `json:"properties,omitempty"`
}

// JsonSchemaType https://json-schema.apifox.cn/#%E6%95%B0%E6%8D%AE%E7%B1%BB%E5%9E%8B
type JsonSchemaType string

const (
	JsonSchemaTypeObject  JsonSchemaType = "object"
	JsonSchemaTypeInteger JsonSchemaType = "integer"
	JsonSchemaTypeString  JsonSchemaType = "string"
	JsonSchemaTypeArray   JsonSchemaType = "array"
	JsonSchemaTypeNumber  JsonSchemaType = "number"
	JsonSchemaTypeBoolean JsonSchemaType = "boolean"
	JsonSchemaTypeNull    JsonSchemaType = "null"
)

type JsonSchemaDefault map[string]any

func testJsonSchema() {

	obj := &JsonSchema{
		Title:   "Person",
		Type:    "object",
		Default: nil,
		Items:   nil,
		Enum:    nil,
		Properties: map[string]*JsonSchema{
			"name": {
				Type:        "string",
				Description: "First and Last name",
				Default:     "Jeremy Dorn",
				MinLength:   4,
			},

			"age": {
				Type:    "integer",
				Default: 25,
				MiniNum: 18,
				MaxiNum: 99,
			},
			"favorite_color": {
				Title:       "favorite color",
				Type:        "string",
				Description: "",
				Default:     "#ffa500",
				Format:      "color",
			},

			"gender": {
				Type: "string",
				Enum: []string{
					"male", "female",
				},
			},

			"pets": {
				Title:       "Pets",
				Type:        "array",
				UniqueItems: true,
				Items: &JsonSchema{
					Title: "Pet",
					Type:  "object",
					Default: JsonSchemaDefault{
						"type": "dog",
						"name": "Walter",
					},
					Properties: map[string]*JsonSchema{
						"type": {
							Type:    "string",
							Default: "dog",
							Enum: []string{
								"cat",
								"dog",
								"bird",
								"reptile",
								"other",
							},
						},
						"name": {
							Type: "string",
						},
					},
				},
			},
		},
	}

	if data, err := json.MarshalIndent(obj, "", "  "); err != nil {
		log.Println(err)
	} else {
		log.Println(string(data))
	}
}
