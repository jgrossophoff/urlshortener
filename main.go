package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

type URLs struct {
	mu   sync.Mutex
	urls map[int]string
}

var (
	listenAddr string
)

func main() {
	flag.StringVar(&listenAddr, "l", ":6868", "Listen address")
	flag.Parse()

	u := &URLs{urls: make(map[int]string)}

	fmt.Printf("listening on %s\n", listenAddr)
	http.ListenAndServe(listenAddr, u)
}

func (u *URLs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if r.Method == "POST" {
		req := struct {
			URL string `json:"url"`
		}{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
			http.Error(w, "bad request", 400)
			return
		}

		_, err := url.Parse(req.URL)
		if err != nil {
			http.Error(w, "bad request", 400)
			return
		}

		idx := len(u.urls) + 1
		if idx == math.MaxInt64 {
			u.urls = make(map[int]string)
		}

		u.urls[idx] = req.URL

		fmt.Fprintf(w, shortURL(r, idx))
	} else if r.Method == "GET" {
		if r.URL.Path == "/clear" {
			u.clear(w, r)
			return
		}
		if r.URL.Path == "" || r.URL.Path == "/" {
			u.renderUI(w, r)
			return
		}

		idx, err := strconv.ParseInt(strings.TrimLeft(r.URL.Path, "/"), 10, 64)
		if err != nil {
			http.Error(w, "bad request", 400)
			return
		}

		url, exists := u.urls[int(idx)]
		if !exists {
			http.Error(w, "not found", 404)
			return
		}
		http.Redirect(w, r, url, http.StatusSeeOther)
	}
}

func shortURL(r *http.Request, idx int) string {
	var scheme string
	if r.TLS != nil {
		scheme = "https://"
	} else {
		scheme = "http://"
	}
	return fmt.Sprintf("%s%s/%d", scheme, r.Host, idx)
}

func (u *URLs) HTML(r *http.Request) string {
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "<table>")

	for idx, v := range u.urls {
		fmt.Fprintf(b, `<tr><td><a href="%[1]s" target="_blank" rel="noopener">%[1]s</a></td><td> -> </td><td><a href="%[2]s" target="_blank" rel="noopener">%[2]s</a></td></tr>`, shortURL(r, idx), html.EscapeString(v))
	}
	fmt.Fprintf(b, "</table>")
	return b.String()
}

func (u *URLs) clear(w http.ResponseWriter, r *http.Request) {
	u.urls = make(map[int]string)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (u *URLs) renderUI(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	fmt.Fprintf(w, `
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<title></title>
	</head>
	<body>
		<strong>URLs</strong>
		<p>
			<pre>%s</pre>
		</p>
		<form>
		<input placeholder="URL" type="url" id="url-input">
			<button>Submit</button>
		</form>
		<p>
			<a href="/clear">Clear</a>
		</p>
		<script>
			(function() {
				let form = document.querySelector("form")
				let input = document.getElementById("url-input")

				input.focus();
				input.value = "https://";

				form.addEventListener("submit", evt => {
					evt.stopPropagation()
					evt.preventDefault()
					let url = input.value.trim()
					fetch("/", {
						method: "POST",
						cache: 'no-cache',
						body: JSON.stringify({url})
					}).then(resp => {
						if (resp.status === 200) window.location.reload()
						else alert(resp.statusText)
					})
				})
			})()
		</script>
	</body>
	</html>
	`, u.HTML(r))
}
