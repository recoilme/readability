package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

func main() {
	file, err := os.Open("html/0.htm")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	node, err := html.Parse(file)
	if err != nil {
		log.Fatal(err)
	}
	_ = node
	var f func(*html.Node, bool)
	f = func(n *html.Node, printText bool) {
		if printText && n.Type == html.TextNode {
			d := n.Data
			space := regexp.MustCompile(`\s+`)
			s := space.ReplaceAllString(d, " ")
			if s != " " && len(s) > 50 {
				fmt.Printf("%s\n", s)
			}

		}
		printText = printText || (n.Type == html.ElementNode && n.Data == "div")
		if n.Data == "script" {
			printText = false
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c, printText)
		}
	}

	var ff func(*html.Node)
	ff = func(n *html.Node) {
		fmt.Printf("%+v\n", n)
		switch n.Type {
		case html.DocumentNode:
			if n.Parent == nil {
				// start
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					ff(c)
				}
			}
		case html.ElementNode:
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				ff(c)
			}
		}
	}

	n := getBody(node)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			fmt.Printf("%+v\n", c.Data)
		}

	}

	//ff(b)
	/*
		content, err := ioutil.ReadFile("html/1.htm")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("File contents: %s", content)
		doc, err := html.Parse(//strings.NewReader(s))
		if err != nil {
			log.Fatal(err)
		}*/
}

func getBody(doc *html.Node) *html.Node {
	var b *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			b = n
			log.Println("return body")
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
		log.Println("return")
	}
	f(doc)
	return b
}

var defHeaders = make(map[string]string)
var defTimeOut = time.Second * 10

func init() {
	defHeaders["User-Agent"] = "Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)"
	//"Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5X Build/MMB29P) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.96 Mobile Safari/537.36 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
	//"Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)"
	//"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:65.0) Gecko/20100101 Firefox/65.0"
	//"Mozilla/5.0 (iPhone; CPU iPhone OS 12_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.0 Mobile/15E148 Safari/604.1"
	//"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.12; rv:52.0) Gecko/20100101 Firefox/52.0"
	defHeaders["Accept"] = "text/html,application/xhtml+xml,application/xml,application/rss+xml;q=0.9,image/webp,*/*;q=0.8"
	defHeaders["Accept-Language"] = "ru-RU,ru;q=0.8,en-US;q=0.5,en;q=0.3"
	//defHeaders["Accept-Encoding"] = "gzip, deflate, br"
	//	defHeaders["Connection"] = "keep-alive"
}

// Get return bytes from url
func get(geturl string, t time.Duration, ua string) ([]byte, error) {
	if t == 0 {
		t = defTimeOut
	}
	ctx, cncl := context.WithTimeout(context.Background(), t)
	defer cncl()
	q := geturl
	if !strings.HasPrefix(geturl, "http") {
		q = "http://" + geturl
	}
	req, err := http.NewRequest(http.MethodGet, q, nil)
	if err != nil {
		return nil, err
	}
	//Host
	u, err := url.Parse(q)
	if err == nil && len(u.Host) > 2 {
		req.Header.Set("Host", u.Host)
	}
	for k, v := range defHeaders {
		req.Header.Set(k, v)
	}
	if ua != "" {
		req.Header.Del("User-Agent")
		req.Header.Set("User-Agent", ua)
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	//log.Println("header", resp.Header.Get("Content-Type"))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return body, nil
	}
	return nil, fmt.Errorf("Error, statusCode:%d", resp.StatusCode)
}

func feedGet(url string, t time.Duration, ua string) (feed *gofeed.Feed, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in f %v\n", r)
		}
	}()
	if t == 0 {
		t = defTimeOut
	}
	b, err := get(url, t, ua)
	if b == nil || err != nil {
		if err.Error() == "Error, statusCode:403" {
			b, err = get(url, t, "Mozilla/5.0 (iPhone; CPU iPhone OS 12_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.0 Mobile/15E148 Safari/604.1")
			if b == nil || err != nil {
				return feed, err
			}
		} else {
			return feed, err
		}
	}
	fp := gofeed.NewParser()
	feed, err = fp.Parse(bytes.NewReader(b))
	return feed, err
}

func feedItems(url string, t time.Duration, maxItems int) (feed *gofeed.Feed, feeds []*gofeed.Item, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered in f %v\n", r)
		}
	}()
	feed, err = feedGet(url, t, "")
	if err != nil || feed == nil {
		return feed, feeds, err
	}
	if len(feed.Items) == 0 {
		return feed, feeds, errors.New("No items,feed:" + url)
	}
	var last = len(feed.Items) - 1
	if last >= maxItems {
		last = maxItems - 1
	}
	for i, _ := range feed.Items {
		if len(feeds) >= maxItems {
			break
		}
		if feed.Items[last-i].Link != "" {
			log.Println("feedItems Link", feed.Items[last-i].Link)
			feeds = append(feeds, feed.Items[last-i])
		}
	}
	return feed, feeds, nil
}

// GetStr return utf8 string from url
func getStr(geturl string, t time.Duration, ua string) (s, contentType string, err error) {
	if t == 0 {
		t = defTimeOut
	}
	ctx, cncl := context.WithTimeout(context.Background(), t)
	defer cncl()
	q := geturl
	if !strings.HasPrefix(geturl, "http") {
		q = "http://" + geturl
	}
	req, err := http.NewRequest(http.MethodGet, q, nil)
	if err != nil {
		return s, contentType, err
	}
	//Host
	u, err := url.Parse(q)
	if err == nil && len(u.Host) > 2 {
		req.Header.Set("Host", u.Host)
	}
	for k, v := range defHeaders {
		req.Header.Set(k, v)
	}
	if ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return s, contentType, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		contentType = resp.Header.Get("Content-Type")
		utf8, err := charset.NewReader(resp.Body, contentType)
		if err != nil {
			return s, contentType, err
		}
		body, err := ioutil.ReadAll(utf8)
		if err != nil {
			return s, contentType, err
		}
		return string(body), contentType, err
	}
	return s, contentType, fmt.Errorf("Error, statusCode:%d", resp.StatusCode)
}

func parse() {
	s, _, _ := getStr("https://www.adme.ru/tvorchestvo-hudozhniki/20-kadrov-kotorye-dokazyvayut-chto-inogda-smekalka-fotografa-reshaet-vse-2072265/",
		0, "")
	//"https://bash.im/quote/455661", 0, "")
	//s := `<p>Links:</p><ul><li><a href="foo">Foo</a><li><a href="/bar/baz">BarBaz</a></ul>`
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		log.Fatal(err)
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			//log.Printf("TextNode:%+v\n", n)
		case html.DocumentNode:
			if n.Parent == nil {
				// start
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			} else {
				log.Printf("DocumentNode:%+v\n", n)
			}

		case html.ElementNode:
			elem := strings.ToLower(n.Data)
			if elem == "html" {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			} else {
				switch elem {
				case "head":
					head(n)
				case "body":
					body(n)
				}
			}
		}
	}
	f(doc)
}

func head(n *html.Node) {
	log.Println("head")
	var f func(*html.Node, bool)
	f = func(n *html.Node, printText bool) {
		if printText && n.Type == html.TextNode {
			fmt.Printf("%q\n", n.Data)
		}
		printText = printText || (n.Type == html.ElementNode && n.Data == "title")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c, printText)
		}
	}
	f(n, false)
}

func body(n *html.Node) {
	log.Println("body")
	var f func(*html.Node, bool)
	f = func(n *html.Node, printText bool) {
		if printText && n.Type == html.TextNode {
			d := n.Data
			space := regexp.MustCompile(`\s+`)
			s := space.ReplaceAllString(d, " ")
			if s != " " && len(s) > 25 {
				fmt.Printf("%s\n", s)
			}

		}
		printText = printText || (n.Type == html.ElementNode && n.Data == "div")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c, printText)
		}
	}
	f(n, false)
}
