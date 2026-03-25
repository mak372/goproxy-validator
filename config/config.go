package config

import (
	"encoding/json"
	"os"
)

type Contract struct {
	Endpoint string            `json:"endpoint"`
	Method   string            `json:"method"`
	Request  map[string]string `json:"request"`
	Response map[string]string `json:"response"`
}

func LoadContract(path string) (*Contract, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var contract Contract
	err = json.Unmarshal(file, &contract)
	if err != nil {
		return nil, err
	}

	return &contract, nil
}
