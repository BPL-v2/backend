package client

import (
	"bpl/config"
	"bpl/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
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
		return nil, "", fmt.Errorf("failed to marshal character data: %v", err)
	}
	gameVersion := "poe1"
	if characterData.Realm == Poe2 {
		gameVersion = "poe2"
	}
	request, err := http.NewRequest("POST", config.Env().POBServerURL+"/"+gameVersion+"/import-character", bytes.NewReader(jsonData))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %v", err)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("failed to perform request: %v", err)
	}
	defer utils.Closer(response.Body)()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %v", err)
	}
	export := string(body)
	pob, err := DecodePoBExport(export)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode PoB export: %v", err)
	}
	return pob, export, nil
}

// Sites Path of Building itself supports importing from, mirrored from
// buildSites.websiteList in PathOfBuildingCommunity/PathOfBuilding's
// src/Modules/BuildSiteTools.lua so we resolve the exact same links a user
// would paste straight into the desktop app.
type pobShareSite struct {
	label       string
	matchURL    *regexp.Regexp
	downloadURL string // %s is replaced with the captured share ID
}

var pobShareSites = []pobShareSite{
	{"Maxroll", regexp.MustCompile(`^https://maxroll\.gg/poe/pob/([^/?#\s]+)`), "https://maxroll.gg/poe/api/pob/%s"},
	{"pob.codes", regexp.MustCompile(`^https://pob\.codes/b/([^/?#\s]+)`), "https://api.pob.codes/%s/raw"},
	{"pobb.in", regexp.MustCompile(`^https://pobb\.in/([^/?#\s]+)`), "https://pobb.in/pob/%s"},
	{"PoE Ninja", regexp.MustCompile(`^https://poe\.ninja/(?:poe1/)?pob/(\w+)`), "https://poe.ninja/poe1/pob/raw/%s"},
	{"Pastebin.com", regexp.MustCompile(`^https://pastebin\.com/(\w+)`), "https://pastebin.com/raw/%s"},
	{"PastebinP.com", regexp.MustCompile(`^https://pastebinp\.com/(\w+)`), "https://pastebinp.com/raw/%s"},
	{"Rentry.co", regexp.MustCompile(`^https://rentry\.co/(\w+)`), "https://rentry.co/paste/%s/raw"},
	{"poedb.tw", regexp.MustCompile(`^https://poedb\.tw/pob/([^/?#\s]+)`), "https://poedb.tw/pob/%s/raw"},
}

// FetchPoBFromShareLink resolves a share link from one of the sites Path of
// Building supports importing from (see pobShareSites) into its raw PoB
// export code. Returns an error if the link doesn't match any known site.
func FetchPoBFromShareLink(shareURL string) (string, error) {
	for _, site := range pobShareSites {
		match := site.matchURL.FindStringSubmatch(shareURL)
		if match == nil {
			continue
		}
		downloadURL := fmt.Sprintf(site.downloadURL, match[1])
		request, err := http.NewRequest("GET", downloadURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %v", err)
		}
		request.Header.Set("User-Agent", "bpl-pob-import")
		response, err := httpClient.Do(request)
		if err != nil {
			return "", fmt.Errorf("failed to fetch PoB code from %s: %v", site.label, err)
		}
		defer utils.Closer(response.Body)()
		if response.StatusCode != 200 {
			return "", fmt.Errorf("%s returned status %d", site.label, response.StatusCode)
		}
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response body: %v", err)
		}
		return strings.TrimSpace(string(body)), nil
	}
	return "", fmt.Errorf("unrecognized PoB share link")
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
