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
	username     string
	password     string
	client       *http.Client
	LastResponse *http.Response
	BasicAuth    bool
}

func NewBasicAuthClient(surl, username, password string) (*Client, error) {
	if username == "" {
		return nil, errors.New("username is empty")
	}

	if password == "" {
		return nil, errors.New("password is empty")
	}

	nurl, err := url.Parse(surl)
	if err != nil {
		return nil, err
	}

	if !nurl.IsAbs() {
		return nil, errors.New("URL is not absolute")
	}

	return &Client{url: nurl, BasicAuth: true, username: username, password: password, client: &http.Client{}}, nil
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
		fw, err := w.CreateFormFile(field, filepath.Base(file))
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(fw, f)
		if err != nil {
			return nil, err
		}
		f.Close()
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

func (c *Client) GetResponse(r *http.Request) (res *http.Response, err error) {
	if c.BasicAuth {
		r.SetBasicAuth(c.username, c.password)
	}

	res, err = c.client.Do(r)
	if err != nil {
		return nil, err
	}
	c.LastResponse = res
	return res, nil
}

func (c *Client) GetJson(uri string, data interface{}) (err error) {
	req, err := c.MakeRequest("GET", uri)
	if err != nil {
		return err
	}

	return c.jsonResponse(req, &data)
}

func (c *Client) PostMultipartJson(uri string, mpf MultipartForm, data interface{}) (err error) {
	req, err := c.MakeMultipartRequest("POST", uri, mpf)
	if err != nil {
		return err
	}

	return c.jsonResponse(req, &data)
}

func (c *Client) jsonResponse(req *http.Request, data interface{}) (err error) {
	res, err := c.GetResponse(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, &data)
}
