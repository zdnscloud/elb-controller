package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	failedRetries  = 5
	failedWaitTime = 5 * time.Second
)

func get(url, token string, obj interface{}) error {
	method := http.MethodGet

	resp, err := sendRequest(method, url, token, bytes.NewBuffer([]byte{}))
	if err != nil {
		return formatError(method, url, err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return formatError(method, url, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		if err := json.Unmarshal(body, obj); err != nil {
			return formatError(method, url, err)
		}
		return nil
	case http.StatusCreated:
		return nil
	case http.StatusNoContent:
		return nil
	default:
		errInfo := struct {
			Message string `json:"message"`
		}{}
		json.Unmarshal(body, &errInfo)
		return fmt.Errorf("%s %s failed with status %v : %s", method, url, resp.StatusCode, errInfo.Message)
	}
}

func create(url, token string, obj interface{}) error {
	method := http.MethodPost

	reqBody, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return formatError(method, url, err)
	}

	resp, err := sendRequest(method, url, token, bytes.NewBuffer(reqBody))
	if err != nil {
		return formatError(method, url, err)
	}

	defer resp.Body.Close()
	return checkRequestResult(method, url, resp.StatusCode, resp.Body)
}

func update(url, token string, obj interface{}) error {
	method := http.MethodPut

	reqBody, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return formatError(method, url, err)
	}

	resp, err := sendRequest(method, url, token, bytes.NewBuffer(reqBody))
	if err != nil {
		return formatError(method, url, err)
	}

	defer resp.Body.Close()
	return checkRequestResult(method, url, resp.StatusCode, resp.Body)
}

func delete(url, token string) error {
	method := http.MethodDelete

	resp, err := sendRequest(method, url, token, bytes.NewBuffer([]byte{}))
	if err != nil {
		return formatError(method, url, err)
	}

	defer resp.Body.Close()
	return checkRequestResult(method, url, resp.StatusCode, resp.Body)
}

func actionWithRetry(url, token string) error {
	var err error
	for i := 0; i < failedRetries; i++ {
		err = action(url, token)
		if err == nil {
			return nil
		}
		time.Sleep(failedWaitTime)
	}
	return err
}

func action(url, token string) error {
	method := http.MethodPost

	resp, err := sendRequest(method, url, token, bytes.NewBuffer([]byte{}))
	if err != nil {
		return formatError(method, url, err)
	}

	defer resp.Body.Close()
	return checkRequestResult(method, url, resp.StatusCode, resp.Body)
}

func sendRequest(method, url, token string, reqBody io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+token)

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	return cli.Do(req)
}

func checkRequestResult(method, url string, code int, body io.ReadCloser) error {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return formatError(method, url, err)
	}

	switch code {
	case http.StatusOK:
		return nil
	case http.StatusCreated:
		return nil
	case http.StatusNoContent:
		return nil
	default:
		errInfo := struct {
			Message string `json:"message"`
		}{}
		json.Unmarshal(b, &errInfo)
		return fmt.Errorf("%s %s failed with status %v : %s", method, url, code, errInfo.Message)
	}
}

func formatError(method, url string, e error) error {
	return fmt.Errorf("%s %s failed %s", method, url, e.Error())
}
