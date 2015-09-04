package main

import (
	"errors"
	"golang.org/x/net/html"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func redirectFunc(req *http.Request, via []*http.Request) error {
	if len(via) > 10 {
		return errors.New("too many redirects")
	}
	req.Header = via[0].Header
	return nil
}

func RequestURL(url, method, username, password string) *http.Response {
	log.Printf("requesting %s.\n", url)

	client := &http.Client{
		CheckRedirect: redirectFunc,
	}

	var res *http.Response
	var err error
	if username == "" {
		res, err = client.Get(url)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("setting basic auth to %s:%s\n", username, password)
		req.SetBasicAuth(username, password)

		res, err = client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
	}

	if res.StatusCode == 401 {
		log.Fatalf("status code %d for %s: auth challenge is [%s]\n", res.StatusCode, url, res.Header.Get("WWW-Authenticate"))
	}
	if res.StatusCode != 200 {
		log.Fatalf("status code %d for %s\n", res.StatusCode, url)
	}

	return res
}

func OpenURL(url, username, password string) io.ReadCloser {
	return RequestURL(url, "GET", username, password).Body
}

func SizeURL(url, username, password string) uint64 {
	size, _ := strconv.ParseUint(
		RequestURL(url, "HEAD", username, password).Header.Get("Content-Length"),
		10, 64,
	)
	return size
}

func ReadURL(url, username, password string) []byte {
	body := OpenURL(url, username, password)
	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("got %d bytes for %s\n", len(data), url)
	return data
}

func SaveBytes(data []byte, dst string) {
	file, err := os.Create(dst)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	n, err := file.Write(data)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("saved %d bytes into %s\n", n, dst)
}

func SaveURL(url, username, password, dst string, file_size uint64, total_done uint64, total_size uint64) {
	log.Printf("%s -> %s\n", url, dst)

	reader := OpenURL(url, username, password)
	defer reader.Close()

	writer, err := os.Create(dst)
	if err != nil {
		log.Fatal(err)
	}
	defer writer.Close()

	var file_done uint64 = 0
	for {
		written, err := io.CopyN(writer, reader, 102400)
		file_done += uint64(written)
		log.Printf(
			"fetching %s (%d/%d bytes of file = %0.2f%%, %d/%d bytes total = %0.2f%%)\n",
			dst,
			file_done, file_size, float32(file_done) / float32(file_size),
			total_done + file_done, total_size, float32(total_done + file_done) / float32(total_size),
		)
		if err != nil {
			break
		}
	}
}

func ExtractMboxURLs(n *html.Node) []string {
	urls := make([]string, 0)
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" && strings.Contains(attr.Val, "mbox") {
				urls = append(urls, attr.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		urls = append(urls, ExtractMboxURLs(c)...)
	}
	return urls
}

type NameSize struct {
	Name string
	Size uint64
}

func main() {
	url := "http://postgresql.org/list/pgsql-hackers/"
	reader := OpenURL(url, "", "")
	defer reader.Close()
	doc, err := html.Parse(reader)
	if err != nil {
		log.Fatalf("could not parse %s", url)
	}
	urls := ExtractMboxURLs(doc)

	sizes := make(map[string]uint64)
	sizechan := make(chan NameSize)
	for _, url := range urls {
		go func(url string) {
			size := SizeURL(
				"http://postgresql.org" + url,
				"archives",
				"antispam",
			)
			log.Printf("%d : %s\n", size, url)
			sizechan <- NameSize{url, size}
		}(url)
	}
	for i := 0; i < len(urls); i++ {
		namesize := <-sizechan
		sizes[namesize.Name] = namesize.Size
	}

	var total_size uint64 = 0
	for url, size := range sizes {
		total_size += size
		log.Printf("%d = %s\n", size, url)
	}

	log.Printf("%d bytes total\n", total_size)

	var total_done uint64 = 0
	for _, url := range urls {
		file_name := filepath.Base(url)
		file_size := sizes[url]
		SaveURL(
			"http://postgresql.org" + url,
			"archives",
			"antispam",
			file_name,
			file_size,
			total_done,
			total_size,
		)
		total_done += file_size
	}
	log.Printf("done.\n")
}
