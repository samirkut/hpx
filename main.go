package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/elazarl/goproxy"
	"golang.org/x/term"
)

func main() {
	var err error
	var verbose bool
	var proxyServer string
	var listenAddr string

	var proxyURL *url.URL
	var useNtlmAuth bool
	var proxyUsername, proxyPassword, proxyDomain string

	flag.BoolVar(&verbose, "verbose", false, "verbose mode")

	flag.StringVar(&proxyServer, "proxy", "", "cascading proxy server")
	flag.BoolVar(&useNtlmAuth, "ntlm", false, "use ntlm auth")
	flag.StringVar(&listenAddr, "addr", "localhost:8080", "proxy listen addr")

	flag.StringVar(&proxyUsername, "user", "", "username for proxy auth")
	flag.StringVar(&proxyDomain, "domain", "", "domain for proxy auth")
	flag.StringVar(&proxyPassword, "password", "", "password for proxy auth")

	flag.Parse()

	if verbose {
		debugf = verboseDebug
	}

	if useNtlmAuth {
		if proxyUsername != "" && proxyPassword == "" {
			//prompt for password if username is provided but not the password
			fmt.Print("Password: ")

			var data []byte
			data, err = term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				log.Fatal(err)
			}
			proxyPassword = string(data)
		}
	}

	proxyURL, err = url.Parse(proxyServer)
	if err != nil {
		log.Fatal(err)
	}

	gpx := goproxy.NewProxyHttpServer()
	gpx.Verbose = true

	if proxyServer != "" {
		gpx.Tr.DialContext = NewDialContext(proxyURL, useNtlmAuth, proxyUsername, proxyPassword, proxyDomain)
		gpx.ConnectDial = NewConnectDial(proxyURL, useNtlmAuth, proxyUsername, proxyPassword, proxyDomain)
	}

	log.Printf("Listening on %s", listenAddr)
	if err = http.ListenAndServe(listenAddr, gpx); err != nil {
		log.Fatal(err)
	}
}

func NewDialContext(proxyURL *url.URL, useNtlmAuth bool, proxyUsername, proxyPassword, proxyDomain string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialProxy := func() (net.Conn, error) {
			debugf("ntlm> Will connect to proxy at " + proxyURL.Host)
			if proxyURL.Scheme == "https" {
				return tls.DialWithDialer(dialer, "tcp", proxyURL.Host, nil)
			}
			return dialer.DialContext(ctx, network, proxyURL.Host)
		}

		if !useNtlmAuth {
			return dialProxy()
		}

		if proxyUsername == "" {
			return dialAndNegotiateAuto(addr, dialProxy)
		}

		return dialAndNegotiate(addr, proxyUsername, proxyPassword, proxyDomain, dialProxy)
	}
}

func NewConnectDial(proxyURL *url.URL, useNtlmAuth bool, proxyUsername, proxyPassword, proxyDomain string) func(network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	return func(network, addr string) (net.Conn, error) {
		dialProxy := func() (net.Conn, error) {
			debugf("ntlm> Will connect to proxy at " + proxyURL.Host)
			if proxyURL.Scheme == "https" {
				return tls.DialWithDialer(dialer, "tcp", proxyURL.Host, nil)
			}
			return dialer.Dial(network, proxyURL.Host)
		}

		if !useNtlmAuth {
			return dialProxy()
		}

		if proxyUsername == "" {
			return dialAndNegotiateAuto(addr, dialProxy)
		}

		return dialAndNegotiate(addr, proxyUsername, proxyPassword, proxyDomain, dialProxy)
	}
}
