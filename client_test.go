package relax

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var goodUrl = "https://ms.example.com"
var user = "user"
var pass = "pass"

type responseHandler struct {
	Message string
	Path    string
}

func (rh responseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != rh.Path {
		http.NotFound(w, r)
	} else {
		w.Write([]byte(rh.Message))
	}
}

func newBasicAuthClientOrFatal(t *testing.T, url, username, password string) *Client {
	c, err := NewBasicAuthClient(url, username, password)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	return c
}

func newClientQueryOrError(t *testing.T, url, uri string) (*Client, string) {
	c := newBasicAuthClientOrFatal(t, url, user, pass)
	query, err := c.GetQuery(uri)
	if err != nil {
		t.Fatalf("Unexpected failure %v", err)
	}
	return c, query
}

func TestNewBasicAuthClient(t *testing.T) {
	c := newBasicAuthClientOrFatal(t, goodUrl, user, pass)

	if c.url.String() != goodUrl {
		t.Errorf("Unexpected URL %v", c.url)
	}

	if c.username != user {
		t.Errorf("username was not set. Expected \"username\", got \"%s\"", c.username)
	}

	if c.password != pass {
		t.Errorf("password was not set. Expected \"password\", got \"%s\"", c.password)
	}

	if c.client == nil {
		t.Fatalf("client was not set")
	}
}

func TestNewBasicAuthClientWithBadUrl(t *testing.T) {
	_, err := NewBasicAuthClient("badurl", user, pass)

	if err == nil {
		t.Errorf("Expected badurl to fail")
	}
}

func TestNewBasicAuthClientWithBadUsername(t *testing.T) {
	_, err := NewBasicAuthClient(goodUrl, "", pass)

	if err == nil {
		t.Fatalf("Expected empty username to fail")
	}

	if err.Error() != "username is empty" {
		t.Errorf("Expected 'username is not empty', got %q", err)
	}
}

func TestNewBasicAuthClientWithBadPassword(t *testing.T) {
	_, err := NewBasicAuthClient(goodUrl, user, "")

	if err == nil {
		t.Fatalf("Expected empty password to fail")
	}

	if err.Error() != "password is empty" {
		t.Errorf("Expected 'password is not empty', got %q", err)
	}
}

func TestGetQuery(t *testing.T) {
	_, query := newClientQueryOrError(t, goodUrl, "/api/foo")
	goodQuery := fmt.Sprintf("%s/api/foo", goodUrl)

	if query != goodQuery {
		t.Errorf("Expected %s, got %s", goodQuery, query)
	}
}

func TestGetQueryWithAbsoluteURI(t *testing.T) {
	c := newBasicAuthClientOrFatal(t, goodUrl, user, pass)
	query, err := c.GetQuery(goodUrl)
	if err == nil {
		t.Fatalf("Expected error, got %v", query)
	}
}

func TestGetResponse(t *testing.T) {
	handler := responseHandler{Message: "test", Path: "/api/foo"}
	server := httptest.NewServer(handler)

	defer server.Close()
	c := newBasicAuthClientOrFatal(t, server.URL, user, pass)
	request, err := c.MakeRequest("GET", "/api/foo")

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

func TestGetJson(t *testing.T) {
	handler := responseHandler{Message: "{\"Foo\": \"bar\"}", Path: "/api/foo"}
	server := httptest.NewServer(handler)

	defer server.Close()
	c := newBasicAuthClientOrFatal(t, server.URL, user, pass)

	var data Response

	err := c.GetJson("/api/foo", &data)

	if err != nil {
		t.Fatal(err)
	}

	if data.Foo != "bar" {
		t.Errorf("Expected data.Foo to be \"bar\", got \"%s\"", data.Foo)
	}

}
