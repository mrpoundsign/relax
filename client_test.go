// Copyright (c) 2014 Brian Nelson. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

var goodURL = "https://ms.example.com"
var apiKey = "apiKey"

type responseHandler struct {
	Message             string
	Path                string
	Method              string
	ExpectedBody        string
	ExpectedContentType string
}

func (rh responseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != rh.Path {
		http.NotFound(w, r)
		return
	}

	if r.Method != rh.Method {
		http.Error(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}

	if rh.ExpectedBody != "" {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("error reading body: %s", err), http.StatusBadRequest)
			return
		}

		if string(body) != rh.ExpectedBody {
			http.Error(w, fmt.Sprintf("body \"%s\" does not match \"%s\"", body, rh.ExpectedBody), http.StatusBadRequest)
			return
		}
	}

	_, err := w.Write([]byte(rh.Message))
	if err != nil {
		http.Error(w, fmt.Sprintf("could not write response: %s", err), http.StatusBadRequest)
	}
}

func newClientOrFatal(t *testing.T, url, apiKey string) *Client {
	c, err := NewClient(url, apiKey)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	return c
}

func newClientQueryOrError(t *testing.T, url, uri string) (*Client, string) {
	c := newClientOrFatal(t, url, apiKey)
	query, err := c.GetQuery(uri)
	if err != nil {
		t.Fatalf("Unexpected failure %v", err)
	}
	return c, query
}

func TestClient_NewClient(t *testing.T) {
	c := newClientOrFatal(t, goodURL, apiKey)

	if c.url.String() != goodURL {
		t.Errorf("Unexpected URL %v", c.url)
	}

	if c.apiKey != apiKey {
		t.Errorf("apiKey was not set. Expected \"%s\", got \"%s\"", apiKey, c.apiKey)
	}

	if c.client == nil {
		t.Fatalf("client was not set")
	}
}

func TestClient_NewBasicAuthClientWithBadUrl(t *testing.T) {
	_, err := NewClient("badurl", apiKey)

	if err == nil {
		t.Errorf("Expected badurl to fail")
	}
}

func TestClient_NewClientWithBadApiKey(t *testing.T) {
	_, err := NewClient(goodURL, "")

	if err == nil {
		t.Fatalf("Expected bad api key to fail")
	}

	if err.Error() != "api key is empty" {
		t.Errorf("Expected 'api key is empty', got %q", err)
	}
}

func TestClient_GetQuery(t *testing.T) {
	_, query := newClientQueryOrError(t, goodURL, "/api/foo")
	goodQuery := fmt.Sprintf("%s/api/foo", goodURL)

	if query != goodQuery {
		t.Errorf("Expected %s, got %s", goodQuery, query)
	}
}

func TestClient_GetQueryWithAbsoluteURI(t *testing.T) {
	c := newClientOrFatal(t, goodURL, apiKey)
	query, err := c.GetQuery(goodURL)
	if err == nil {
		t.Fatalf("Expected error, got %v", query)
	}
}

func TestClient_GetResponse(t *testing.T) {
	handler := responseHandler{Method: http.MethodGet, Message: "test", Path: "/api/foo"}
	server := httptest.NewServer(handler)

	defer server.Close()
	c := newClientOrFatal(t, server.URL, apiKey)
	request, err := c.MakeRequest(http.MethodGet, "/api/foo")

	if err != nil {
		t.Fatal(err)
	}
	res, err := c.GetResponse(request)

	if err != nil {
		t.Fatal(err)
	}

	got, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "test" {
		t.Errorf("got %q, want test", string(got))
	}
}

type Response struct {
	Foo string
}

func TestClient_CreateJson(t *testing.T) {
	type postData struct {
		Name string
	}
	handler := responseHandler{Method: http.MethodPost, Message: "{\"Foo\": \"bar\"}", Path: "/api/foo", ExpectedBody: "{\"Name\":\"new_name\"}"}
	server := httptest.NewServer(handler)

	defer server.Close()
	c := newClientOrFatal(t, server.URL, apiKey)

	var response Response
	data := postData{Name: "new_name"}

	err := c.CreateJson("/api/foo", data, &response)

	if err != nil {
		t.Fatal(err)
	}

	if response.Foo != "bar" {
		t.Errorf("Expected data.Foo to be \"bar\", got \"%s\"", response.Foo)
	}

	err = c.CreateJson("/api/foo", data, nil)

	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_UpdateJson(t *testing.T) {
	type postData struct {
		Name string
	}
	handler := responseHandler{Method: http.MethodPut, Message: "{\"Foo\": \"bar\"}", Path: "/api/foo", ExpectedBody: "{\"Name\":\"new_name\"}"}
	server := httptest.NewServer(handler)

	defer server.Close()
	c := newClientOrFatal(t, server.URL, apiKey)

	var response Response
	data := postData{Name: "new_name"}

	err := c.UpdateJson("/api/foo", data, &response)

	if err != nil {
		t.Fatal(err)
	}

	if response.Foo != "bar" {
		t.Errorf("Expected data.Foo to be \"bar\", got \"%s\"", response.Foo)
	}

	err = c.UpdateJson("/api/foo", data, nil)

	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_ReadJson(t *testing.T) {
	handler := responseHandler{Method: http.MethodGet, Message: "{\"Foo\": \"bar\"}", Path: "/api/foo"}
	server := httptest.NewServer(handler)

	defer server.Close()
	c := newClientOrFatal(t, server.URL, apiKey)

	var data Response

	err := c.ReadJson("/api/foo", &data)

	if err != nil {
		t.Fatal(err)
	}

	if data.Foo != "bar" {
		t.Errorf("Expected data.Foo to be \"bar\", got \"%s\"", data.Foo)
	}

	err = c.ReadJson("/api/foo", nil)

	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_DeleteJson(t *testing.T) {
	handler := responseHandler{Method: http.MethodDelete, Message: "{\"Foo\": \"bar\"}", Path: "/api/foo"}
	server := httptest.NewServer(handler)

	defer server.Close()
	c := newClientOrFatal(t, server.URL, apiKey)

	var data Response

	err := c.DeleteJson("/api/foo", &data)

	if err != nil {
		t.Fatal(err)
	}

	if data.Foo != "bar" {
		t.Errorf("Expected data.Foo to be \"bar\", got \"%s\"", data.Foo)
	}

	err = c.DeleteJson("/api/foo", nil)

	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_jsonResponse(t *testing.T) {
	type fields struct {
		url          *url.URL
		username     string
		password     string
		client       *http.Client
		LastResponse *http.Response
		LastBody     []byte
		BasicAuth    bool
	}
	type args struct {
		req  *http.Request
		data interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:          tt.fields.url,
				apiKey:       apiKey,
				client:       tt.fields.client,
				LastResponse: tt.fields.LastResponse,
				LastBody:     tt.fields.LastBody,
			}
			if err := c.jsonResponse(tt.args.req, tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("Client.jsonResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
