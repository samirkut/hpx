package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/rapid7/go-get-proxied/proxy"
)

func main() {

	var portPtr = flag.Int("port", 8080, "proxy port")

	flag.Parse()

	gpx := goproxy.NewProxyHttpServer()
	gpx.Verbose = true
	gpx.Tr.Proxy = getSystemProxy

	addr := fmt.Sprintf("localhost:%d", *portPtr)
	log.Printf("Listening on %s", addr)
	if err := http.ListenAndServe(addr, gpx); err != nil {
		log.Fatal(err)
	}
}

var provider = proxy.NewProvider("")

func getSystemProxy(req *http.Request) (*url.URL, error) {
	log.Printf("Get proxy for %s", req.RequestURI)

	scheme := req.URL.Scheme
	if scheme == "" {
		arr := strings.Split(req.RequestURI, ":")
		if len(arr) > 0 {
			scheme = arr[0]
		}
	}

	if scheme == "" {
		scheme = "http"
	}

	proxy := provider.GetProxy(scheme, req.RequestURI)
	if proxy == nil {
		log.Println("Using direct connection")
		return nil, nil // no proxy
	}

	log.Printf("Using %s", proxy.URL())
	return proxy.URL(), nil
}
