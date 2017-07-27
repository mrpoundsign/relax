// Copyright (c) 2014 Brian Nelson. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type Client struct {
	url          *url.URL
	apiKey       string
	client       *http.Client
	LastResponse *http.Response
	LastBody     []byte
}

func NewClient(surl, apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("api key is empty")
	}

	nurl, err := url.Parse(surl)
	if err != nil {
		return nil, err
	}

	if !nurl.IsAbs() {
		return nil, errors.New("URL is not absolute")
	}

	return &Client{url: nurl, client: &http.Client{}, apiKey: apiKey}, nil
}

func (c *Client) GetQuery(uri string) (string, error) {
	nurl, err := url.Parse(uri)

	if err != nil {
		return "", err
	}
	if nurl.IsAbs() {
		return "", errors.New("URI is not absolute")
	}

	return c.url.ResolveReference(nurl).String(), nil
}

func (c *Client) MakeRequest(method, uri string) (*http.Request, error) {
	query, err := c.GetQuery(uri)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(method, query, nil)
	if err != nil {
		return nil, err
	}

	return request, nil
}

func (c *Client) MakeMultipartRequest(method, uri string, mpf MultipartForm) (req *http.Request, err error) {
	query, err := c.GetQuery(uri)
	if err != nil {
		return nil, err
	}

	b := &bytes.Buffer{}

	w := multipart.NewWriter(b)

	// Add files
	for field, file := range mpf.Files {
		f, err := os.Open(file)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error with file %s, %s", file, err.Error()))
		}
		defer f.Close()

		fw, err := w.CreateFormFile(field, filepath.Base(file))
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(fw, f)
		if err != nil {
			return nil, err
		}
	}

	for field, value := range mpf.Fields {
		err := w.WriteField(field, value)
		if err != nil {
			return nil, err
		}
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	req, err = http.NewRequest(method, query, b)
	if err != nil {
		return req, err
	}

	req.Header.Add("Content-Type", w.FormDataContentType())

	return
}

func (c *Client) PostMultipartJson(uri string, mpf MultipartForm, data interface{}) (err error) {
	req, err := c.MakeMultipartRequest(http.MethodPost, uri, mpf)
	if err != nil {
		return err
	}

	return c.jsonResponse(req, &data)
}

func (c *Client) GetResponse(r *http.Request) (res *http.Response, err error) {
	if c.apiKey != "" {
		r.Header.Set("Autorization", fmt.Sprintf("Token token=\"%s\"", c.apiKey))
	}

	res, err = c.client.Do(r)
	if err != nil {
		return nil, err
	}
	c.LastResponse = res

	return res, nil
}

func (c *Client) ReadJson(uri string, response interface{}) (err error) {
	req, err := c.MakeRequest(http.MethodGet, uri)
	if err != nil {
		return err
	}

	return c.jsonResponse(req, &response)
}

func (c *Client) DeleteJson(uri string, response interface{}) (err error) {
	req, err := c.MakeRequest(http.MethodDelete, uri)
	if err != nil {
		return err
	}

	return c.jsonResponse(req, &response)
}

func (c *Client) CreateJson(uri string, data interface{}, response interface{}) (err error) {
	req, err := c.MakeRequest(http.MethodPost, uri)
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application.json")
	req.Body = ioutil.NopCloser(bytes.NewReader(jsonData))

	return c.jsonResponse(req, &response)
}

func (c *Client) UpdateJson(uri string, data interface{}, response interface{}) (err error) {
	req, err := c.MakeRequest(http.MethodPut, uri)
	if err != nil {
		return err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = ioutil.NopCloser(bytes.NewReader(jsonData))

	return c.jsonResponse(req, &response)
}

func (c *Client) jsonResponse(req *http.Request, response interface{}) (err error) {
	if response == nil {
		return nil
	}

	res, err := c.GetResponse(req)
	if err != nil {
		return err
	}

	c.LastBody, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(c.LastBody, &response)

	if err != nil {
		return fmt.Errorf("Invalid JSON: %s", c.LastBody)
	}

	return nil
}
