package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"

	"github.com/elazarl/goproxy"
	"github.com/rapid7/go-get-proxied/proxy"
)

func main() {

	var portPtr = flag.Int("port", 8080, "proxy port")

	flag.Parse()

	gpx := goproxy.NewProxyHttpServer()
	gpx.Verbose = true
	gpx.ConnectDialWithReq = func(req *http.Request, network, addr string) (net.Conn, error) {
		upstreamProxy, err := GetSystemProxy(req.RequestURI)
		if err != nil {
			return nil, err
		}

		if upstreamProxy == "" {
			if gpx.Tr.Dial != nil {
				return gpx.Tr.Dial(network, addr)
			}
			return net.Dial(network, addr)
		}

		fn := gpx.NewConnectDialToProxy(upstreamProxy)
		return fn(network, addr)
	}

	addr := fmt.Sprintf("localhost:%d", *portPtr)
	log.Printf("Listening on %s", addr)
	if err := http.ListenAndServe(addr, gpx); err != nil {
		log.Fatal(err)
	}
}

var provider = proxy.NewProvider("")

func GetSystemProxy(targetUrl string) (string, error) {
	log.Printf("Get proxy for %s", targetUrl)
	u, err := url.Parse(targetUrl)
	if err != nil {
		return "", err
	}

	proxy := provider.GetProxy(u.Scheme, targetUrl)
	if proxy == nil {
		log.Println("Using direct connection")
		return "", nil // no proxy
	}

	log.Printf("Using %s", proxy.URL())
	return proxy.URL().String(), nil
}
