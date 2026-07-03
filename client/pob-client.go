package client

import (
	"bpl/config"
	"bpl/utils"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// Shared HTTP client with optimized transport for multiple replicas
var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10, // Allow up to 10 connections per host (covers all replicas)
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	},
	Timeout: 30 * time.Second,
}

func GetPoBExport(characterData *Character) (*PathOfBuilding, string, error) {
	jsonData, err := json.Marshal(characterData)
	if err != nil {
		return nil, "", err
	}
	gameVersion := "poe1"
	if characterData.Realm == PoE2 {
		gameVersion = "poe2"
	}
	request, err := http.NewRequest("POST", config.Env().POBServerURL+"/"+gameVersion+"/import-character", bytes.NewReader(jsonData))
	if err != nil {
		return nil, "", err
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer utils.Closer(response.Body)()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", err
	}
	export := string(body)
	pob, err := DecodePoBExport(export)
	if err != nil {
		return nil, "", err
	}
	return pob, export, nil
}

func UpdatePoBExport(pobString string) (string, error) {
	// send text to pob server and get updated export
	gameVersion := "poe1"
	request, err := http.NewRequest("POST", config.Env().POBServerURL+"/"+gameVersion+"/update-config", bytes.NewReader([]byte(pobString)))
	if err != nil {
		return "", err
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer utils.Closer(response.Body)()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	updatedExport := string(body)
	return updatedExport, nil
}
