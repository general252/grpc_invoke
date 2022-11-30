package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var (
	_defaultConfig = newConfig()
)

func GetConfig() *config {
	return _defaultConfig
}

type config struct {
	Services []Service `json:"services"`

	filename string
}

func newConfig() *config {
	dir, _ := GetExeDir()
	filename := fmt.Sprintf("%v/config.json", dir)

	tis := &config{
		filename: filename,
		Services: []Service{},
	}

	tis.Load()
	tis.Storage()

	return tis
}

func (tis *config) Load() {
	data, err := os.ReadFile(tis.filename)
	if err != nil {
		return
	}

	_ = json.Unmarshal(data, tis)
}

func (tis *config) Storage() {
	data, err := json.MarshalIndent(tis, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(tis.filename, data, os.ModePerm)
}

func GetExeDir() (string, error) {
	dir := filepath.Dir(os.Args[0])
	absDir, err := filepath.Abs(dir)

	return absDir, err
}

type Service struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}
