package main

import (
	"net/http"
	"fmt"
	"log"
	"io"
	"io/ioutil"
	"golang.org/x/net/html"
	"strings"
	"path/filepath"
)

func HTTPDownloadBytes(uri string) ([]byte, error) {
	fmt.Printf("HTTPDownload From: %s.\n", uri)
	res, err := http.Get(uri)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	d, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ReadFile: Size of download: %d\n", len(d))
	return d, err
}

func HTTPDownload(uri string) (io.Reader) {
	fmt.Printf("HTTPDownload From: %s.\n", uri)
	res, err := http.Get(uri)
	if err != nil {
		log.Fatal(err)
	}
	return res.Body
}

func WriteFile(dst string, d []byte) error {
	fmt.Printf("WriteFile: Size of download: %d\n", len(d))
	err := ioutil.WriteFile(dst, d, 0444)
	if err != nil {
		log.Fatal(err)
	}
	return err
}

func DownloadToFile(uri string, dst string) {
	fmt.Printf("DownloadToFile From: %s.\n", uri)
	if d, err := HTTPDownloadBytes(uri); err == nil {
		fmt.Printf("downloaded %s.\n", uri)
		if WriteFile(dst, d) == nil {
			fmt.Printf("saved %s as %s\n", uri, dst)
		}
	}
}

func ExtractMboxUris(n *html.Node) []string {
	uris := make([]string, 0)
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" && strings.Contains(attr.Val, "mbox") {
				uris = append(uris, attr.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		uris = append(uris, ExtractMboxUris(c)...)
	}
	return uris
}

func main() {
	uri := "http://postgresql.org/list/pgsql-hackers/"
	reader := HTTPDownload(uri)
	doc, err := html.Parse(reader)
	if err != nil {
		log.Fatal("could not parse %s", uri)
	}
	uris := ExtractMboxUris(doc)
	for _, uri := range(uris) {
		//fmt.Println(uri)
		filename := filepath.Base(uri)
		DownloadToFile("http://postgresql.org" + uri, filename)
	}
}
