package repository

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

func GetManifest(url string) (*Manifest, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status error: %v", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %v", err)
	}

	var manifest Manifest
	if err := xml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}
