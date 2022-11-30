package http_swagger

import (
	"encoding/json"
	"log"
	"os"
)

func ExampleParseSwagger() {
	data, err := os.ReadFile(`F:\Develop\ServerE\bvcr\docs\swagger3.json`)
	if err != nil {
		log.Println(err)
		return
	}
	api, err := ParseSwagger(data)
	if err != nil {
		log.Println(err)
		return
	}

	data, _ = json.MarshalIndent(api, "", "  ")
	log.Println(string(data))

	// output:
}
