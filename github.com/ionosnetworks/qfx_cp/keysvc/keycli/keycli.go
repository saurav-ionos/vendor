package keycli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	hcli "github.com/ionosnetworks/qfx_cmn/httplib/client"
	k "github.com/ionosnetworks/qfx_cp/keysvc/objects"
	kc "github.com/patrickmn/go-cache"
)

const (
	DEFAULT_KEY_EXPIRATION = 300 * time.Second
	CLEANUP_INTERVAL       = 30 * time.Second
)

var (
	enableKeyCheck = false
)

type KeyCli struct {
	SvcKey    string
	SvcSecret string
	KeySvcAP  string
	kcache    *kc.Cache
}

func New(key, secret string) (*KeyCli, error) {
	cli := KeyCli{SvcKey: key, SvcSecret: secret}

	cli.KeySvcAP = "127.0.0.1:9090"
	if val := os.Getenv("KEYSVC_SERVER"); val != "" {
		cli.KeySvcAP = val
	}
	// TODO:: This needs to be removed finally.
	if val := os.Getenv("KEYSVC_ENABLE_CHECK"); val != "" {
		enableKeyCheck = true
	}

	if err := cli.init(); err != nil {
		return nil, err
	}

	return &cli, nil
}

func (keycli *KeyCli) init() error {

	keycli.kcache = kc.New(DEFAULT_KEY_EXPIRATION, CLEANUP_INTERVAL)

	if keycli.SvcKey == "" || keycli.SvcSecret == "" {
		return errors.New("KEY and SECRET cannot be empty")
	}
	// Try getting his own object to verify whether provided key has valid key cred

	if key, err := keycli.Get(keycli.SvcKey); err == nil {

		if keycli.SvcKey != key.AccessKey {
			return errors.New("Invalid Access key")
		}
		if keycli.SvcSecret != key.AccessSecret {
			return errors.New("Invalid Access secret")
		}
	} else {
		return errors.New("Invalid Access Key")
	}

	return nil
}

func (keycli *KeyCli) Get(key string) (k.KeyEntryOut, error) {

	var err error

	// Check if the key is in cache.

	if val, found := keycli.kcache.Get(key); found {

		if keyOut, ok := val.(k.KeyEntryOut); ok {
			return keyOut, nil
		}
		// fmt.Println("This cannot happen")
	}

	cli := hcli.New(keycli.KeySvcAP)
	keyIn := k.KeyIn{AccessKey: key}

	if query, err := json.Marshal(keyIn); err == nil {

		var keyOut k.KeyEntryOut

		if result, err := cli.RunCommand("GET", "key", bytes.NewReader(query)); err == nil {
			if err := json.Unmarshal(result, &keyOut); err == nil {
				keycli.kcache.Set(key, &keyOut, DEFAULT_KEY_EXPIRATION)
				return keyOut, nil
			}
		}
	}

	return k.KeyEntryOut{ErrCode: "Key does not exist"}, err
}

func (keycli *KeyCli) ValidateApiRequest(key, secret, request string) (bool, []string) {

	if enableKeyCheck == false {
		return true, nil
	}

	if keycli == nil || secret == "" || key == "" {
		return false, nil
	}

	keyOut, err := keycli.Get(key)

	if err != nil || keyOut.AccessKey != secret {
		return false, nil
	}

	for i := 0; i < len(keyOut.Credential.ApiList); i++ {
		if strings.HasPrefix("API:"+request, keyOut.Credential.ApiList[i]) {
			return true, keyOut.Credential.FeatureList
		}
	}
	return false, nil
}

func (keycli *KeyCli) ValidateFeatureRequest(key, secret, request string) bool {

	if enableKeyCheck == false {
		return true
	}

	if keycli == nil || secret == "" || key == "" {
		return false
	}

	keyOut, err := keycli.Get(key)

	if err != nil || keyOut.AccessKey != secret {
		return false
	}

	for i := 0; i < len(keyOut.Credential.FeatureList); i++ {
		if request == keyOut.Credential.FeatureList[i] {
			return true
		}
	}
	return false
}
