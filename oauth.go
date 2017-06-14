package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func getAuthenticationURL() string {
	return fmt.Sprintf("http://github.com/login/oauth/authorize?client_id=%s&scope=read:org user", cfg.Github.ClientID)
}

func getAccessToken(code string) (string, error) {
	v := url.Values{}

	v.Set("client_id", cfg.Github.ClientID)
	v.Set("client_secret", cfg.Github.ClientSecret)
	v.Set("code", code)

	buf := bytes.NewReader([]byte(v.Encode()))
	req, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", buf)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	r := map[string]string{}
	return r["access_token"], json.NewDecoder(resp.Body).Decode(&r)
}
