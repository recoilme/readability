package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dyatlov/go-htmlinfo/htmlinfo"
	"github.com/recoilme/readability/web"
)

func main() {
	host := flag.String("host", "localhost", "Host to listen on")
	port := flag.Int("port", 8000, "Port to listen on")
	waitTimeout := flag.Int("wait_timeout", 7, "How much time to wait for/fetch response from remote server")

	flag.Parse()

	startServer(*host, *port, *waitTimeout)
}

func startServer(host string, port int, waitTimeout int) {
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", host, port),
		Handler:        &apiHandler{},
		ReadTimeout:    time.Duration(waitTimeout) * time.Second,
		WriteTimeout:   time.Duration(waitTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}

type apiHandler struct {
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	u := r.FormValue("info")
	if u != "" {
		urlInfo(w, r, u)
		return
	}

	u = r.FormValue("content")
	if u != "" {
		content(w, r, u)
		return
	}
}

func urlInfo(w http.ResponseWriter, r *http.Request, u string) {
	w.Header().Set("Content-Type", "application/json")

	// to be able to retrieve data from javascript directly
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	_, err := url.Parse(u)

	if err != nil {
		renderError(err, w)
		return
	}

	log.Printf("Sending url: %s", u)

	w.WriteHeader(http.StatusOK)
	type res struct {
		URL  string             `json:"url"`
		Info *htmlinfo.HTMLInfo `json:"info"`
	}
	info := getHTMLInfo(u)
	resp := &res{URL: u, Info: info}
	b, err := json.Marshal(resp)
	if err != nil {
		renderError(err, w)
		return
	}
	fmt.Fprintln(w, string(b))
}

func renderError(err error, w http.ResponseWriter) {
	if err != nil {
		http.Error(w, "{\"status\": \"error\", \"message\":\""+err.Error()+"\"}", 422)
		return
	}
}

func getHTMLInfo(url string) *htmlinfo.HTMLInfo {
	info := htmlinfo.NewHTMLInfo()
	b, _ := web.Get(url, 0, "")
	info.Parse(bytes.NewReader(b), &url, nil)
	return info
}

func content(w http.ResponseWriter, r *http.Request, u string) {
	w.Header().Set("Content-Type", "text/html")

	content := web.GetContent(u)
	domain := ""
	parsed, err := url.Parse(u)
	if err == nil {
		domain = parsed.Scheme + "://" + parsed.Host + "/"
	}
	content = strings.Replace(content, "<img src=\"/", "<br/><img src=\""+domain, -1)
	content = strings.Replace(content, "<head>", "<head><meta charset=\"utf-8\"><link rel=\"stylesheet\" href=\"https://cdn.jsdelivr.net/gh/kognise/water.css@latest/dist/dark.min.css\"/>", -1)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, content)

}
