package main

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/pquerna/otp/totp"
	"github.com/prometheus/common/log"
)

type token struct {
	Code   string `json:"code"`
	Icon   string `json:"icon"`
	Name   string `json:"name"`
	Secret string `json:"-"`
}

func (t *token) GenerateCode(in time.Time) error {
	secret := t.Secret

	if n := len(secret) % 8; n != 0 {
		secret = secret + strings.Repeat("=", 8-n)
	}

	var err error
	t.Code, err = totp.GenerateCode(strings.ToUpper(secret), in)
	return err
}

// Sorter interface

type tokenList []*token

func (t tokenList) Len() int           { return len(t) }
func (t tokenList) Less(i, j int) bool { return strings.ToLower(t[i].Name) < strings.ToLower(t[j].Name) }
func (t tokenList) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

func (t tokenList) LongestName() (l int) {
	for _, s := range t {
		if ll := len(s.Name); ll > l {
			l = ll
		}
	}

	return
}

func getSecretsFromVault(accessToken string) ([]*token, error) {
	client, err := api.NewClient(&api.Config{
		Address: cfg.Vault.Address,
	})

	if err != nil {
		return nil, fmt.Errorf("Unable to create client: %s", err)
	}

	if s, err := client.Logical().Write("auth/github/login", map[string]interface{}{"token": accessToken}); err != nil || s.Auth == nil {
		return nil, fmt.Errorf("Login did not work: Error = %s", err)
	} else {
		client.SetToken(s.Auth.ClientToken)
	}

	key := cfg.Vault.Prefix

	resp := []*token{}
	respChan := make(chan *token, 100)

	keyPoolChan := make(chan string, 100)

	scanPool := make(chan string, 100)
	scanPool <- strings.TrimRight(key, "*")

	done := make(chan struct{})
	defer func() { done <- struct{}{} }()

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		for {
			select {
			case key := <-scanPool:
				go scanKeyForSubKeys(client, key, scanPool, keyPoolChan, wg)
			case key := <-keyPoolChan:
				go fetchTokenFromKey(client, key, respChan, wg)
			case t := <-respChan:
				resp = append(resp, t)
			case <-done:
				close(scanPool)
				close(keyPoolChan)
				close(respChan)
				return
			}
		}
	}()

	wg.Wait()

	sort.Sort(tokenList(resp))

	return resp, nil
}

func scanKeyForSubKeys(client *api.Client, key string, subKeyChan, tokenKeyChan chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	s, err := client.Logical().List(key)
	if err != nil {
		log.Errorf("Unable to list keys %q: %s", key, err)
		return
	}

	if s == nil {
		log.Errorf("There is no key %q", key)
		return
	}

	if s.Data["keys"] != nil {
		for _, sk := range s.Data["keys"].([]interface{}) {
			sks := sk.(string)
			if strings.HasSuffix(sks, "/") {
				wg.Add(1)
				subKeyChan <- path.Join(key, sks)
			} else {
				wg.Add(1)
				tokenKeyChan <- path.Join(key, sks)
			}
		}
	}
}

func fetchTokenFromKey(client *api.Client, k string, respChan chan *token, wg *sync.WaitGroup) {
	defer wg.Done()

	data, err := client.Logical().Read(k)
	if err != nil {
		log.Errorf("Unable to read from key %q: %s", k, err)
		return
	}

	if data.Data == nil {
		// Key without any data? Weird.
		return
	}

	tok := &token{
		Icon: "key",
		Name: k,
	}

	for k, v := range data.Data {
		switch k {
		case cfg.Vault.SecretField:
			tok.Secret = v.(string)
			tok.GenerateCode(time.Now())
		case "code":
			tok.Code = v.(string)
		case "name":
			tok.Name = v.(string)
		case "account_name":
			tok.Name = v.(string)
		case "icon":
			tok.Icon = v.(string)
		}
	}

	if tok.Code == "" {
		// Nothing ended in us having a code, does not seem to be something for us
		return
	}

	respChan <- tok
}
