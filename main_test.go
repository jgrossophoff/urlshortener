package main

import (
	"io"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTP(t *testing.T) {
	u := NewURLs()
	ts := httptest.NewServer(u)
	defer ts.Close()

	for _, test := range []struct {
		method           string
		path             string
		reqBody          io.Reader
		statusCode       int
		respBodyContains []string
	}{
		{
			"GET", "", nil, 200, []string{"html", "input"},
		},
	} {
		r := httptest.NewRequest(test.method, ts.URL+test.path, test.reqBody)
		w := httptest.NewRecorder()
		u.ServeHTTP(w, r)

		respBody, err := ioutil.ReadAll(w.Body)
		for _, s := range test.respBodyContains {
			if err != nil {
				t.Errorf("error reading response body when running test request %+v: %s", test, err)
			}

			if !strings.Contains(string(respBody), s) {
				t.Errorf("expected response body %q for test request %+v to contain %s, but it didnt.", string(respBody), test, s)
			}
		}

		if w.Code != test.statusCode {
			t.Errorf("expected status code for test request %+v to be %d, was %d.", test, test.statusCode, w.Code)
		}
	}
}
