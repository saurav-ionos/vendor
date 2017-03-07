package client

import (
	"crypto/rand"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"
)

type HttpCli struct {
	accessPoint string
	client      *http.Client
}

func (httpcli *HttpCli) RunCommand(operation, command string, params io.Reader) ([]byte, error) {

	url := "https://" + httpcli.accessPoint + "/" + command

	req, err := http.NewRequest(operation, url, params)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpcli.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	return body, nil
}

func ConfigTLS() *tls.Config {

	TLSConfig := &tls.Config{}

	TLSConfig.Rand = rand.Reader
	TLSConfig.MinVersion = tls.VersionTLS10
	TLSConfig.SessionTicketsDisabled = false
	TLSConfig.InsecureSkipVerify = true

	return TLSConfig
}

func New(accessPoint string) *HttpCli {
	httpcli := HttpCli{accessPoint: accessPoint}

	tr := &http.Transport{
		DisableCompression: true,
	}
	tr.TLSClientConfig = ConfigTLS()
	httpcli.client = &http.Client{Transport: tr}

	return &httpcli
}
