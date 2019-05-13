package web

import (
	"bytes"
	context "context"
	"errors"
	fmt "fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/dyatlov/go-readability"
	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html/charset"
)

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
func Get(geturl string, t time.Duration, ua string) ([]byte, error) {
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
	b, err := Get(url, t, ua)
	if b == nil || err != nil {
		if err.Error() == "Error, statusCode:403" {
			b, err = Get(url, t, "Mozilla/5.0 (iPhone; CPU iPhone OS 12_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.0 Mobile/15E148 Safari/604.1")
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

func GetContent(u string) string {
	s, _, err := getStr(u, 10*time.Second, "")
	if err != nil {
		log.Println(err)
		return ""
	}
	//log.Println(s)
	doc, err := readability.NewDocument(s)
	doc.WhitelistTags = []string{"p", "a", "img", "pre", "b", "h1", "h2", "h3", "h4", "h5", "h6",
		"blockquote", "hr", "strong", "sup", "ul", "ol", "li", "code", "table", "tr", "td"}
	doc.WhitelistAttrs["img"] = []string{"src", "title"}
	doc.WhitelistAttrs["a"] = []string{"href"}
	doc.MinTextLength = 250
	doc.RetryLength = 250

	if err != nil {
		log.Println("getContent", err)
		return ""
	}
	content := doc.Content()
	content = strings.Replace(content, "\r\n", " ", -1)
	content = strings.Replace(content, "\n", " ", -1)
	content = strings.Replace(content, "src=\"//", "src=http://", -1)
	tabs := regexp.MustCompile(`\t+`)
	content = tabs.ReplaceAllString(content, " ")
	space := regexp.MustCompile(`\s+`)
	content = space.ReplaceAllString(content, " ")

	return content
}
