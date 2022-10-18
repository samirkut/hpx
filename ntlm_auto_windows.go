package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/alexbrainman/sspi/ntlm"
)

func dialAndNegotiateAuto(addr string, baseDial func() (net.Conn, error)) (net.Conn, error) {
	conn, err := baseDial()
	if err != nil {
		debugf("ntlm> Could not call dial context with proxy: %s", err)
		return conn, err
	}

	cred, err := ntlm.AcquireCurrentUserCredentials()
	if err != nil {
		return conn, err
	}
	defer cred.Release()

	secctx, negotiate, err := ntlm.NewClientContext(cred)
	if err != nil {
		return conn, err
	}
	defer secctx.Release()

	// NTLM Step 1: Send Negotiate Message
	debugf("ntlm> NTLM negotiate message: '%s'", base64.StdEncoding.EncodeToString(negotiate))
	header := make(http.Header)
	header.Set("Proxy-Authorization", fmt.Sprintf("NTLM %s", base64.StdEncoding.EncodeToString(negotiate)))
	header.Set("Proxy-Connection", "Keep-Alive")
	connect := &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: header,
	}
	if err := connect.Write(conn); err != nil {
		debugf("ntlm> Could not write negotiate message to proxy: %s", err)
		return conn, err
	}
	debugf("ntlm> Successfully sent negotiate message to proxy")
	// NTLM Step 2: Receive Challenge Message
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, connect)
	if err != nil {
		debugf("ntlm> Could not read response from proxy: %s", err)
		return conn, err
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		debugf("ntlm> Could not read response body from proxy: %s", err)
		return conn, err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusProxyAuthRequired {
		debugf("ntlm> Expected %d as return status, got: %d", http.StatusProxyAuthRequired, resp.StatusCode)
		return conn, errors.New(http.StatusText(resp.StatusCode))
	}
	challenge := strings.Split(resp.Header.Get("Proxy-Authenticate"), " ")
	if len(challenge) < 2 {
		debugf("ntlm> The proxy did not return an NTLM challenge, got: '%s'", resp.Header.Get("Proxy-Authenticate"))
		return conn, errors.New("no NTLM challenge received")
	}
	debugf("ntlm> NTLM challenge: '%s'", challenge[1])
	challengeMessage, err := base64.StdEncoding.DecodeString(challenge[1])
	if err != nil {
		debugf("ntlm> Could not base64 decode the NTLM challenge: %s", err)
		return conn, err
	}
	// NTLM Step 3: Send Authorization Message
	authenticate, err := secctx.Update(challengeMessage)
	if err != nil {
		return conn, err
	}

	debugf("ntlm> NTLM authorization: '%s'", base64.StdEncoding.EncodeToString(authenticate))
	header.Set("Proxy-Authorization", fmt.Sprintf("NTLM %s", base64.StdEncoding.EncodeToString(authenticate)))
	connect = &http.Request{
		Method: "CONNECT",
		URL:    &url.URL{Opaque: addr},
		Host:   addr,
		Header: header,
	}
	if err := connect.Write(conn); err != nil {
		debugf("ntlm> Could not write authorization to proxy: %s", err)
		return conn, err
	}
	resp, err = http.ReadResponse(br, connect)
	if err != nil {
		debugf("ntlm> Could not read response from proxy: %s", err)
		return conn, err
	}
	if resp.StatusCode != http.StatusOK {
		debugf("ntlm> Expected %d as return status, got: %d", http.StatusOK, resp.StatusCode)
		return conn, errors.New(http.StatusText(resp.StatusCode))
	}
	// Succussfully authorized with NTLM
	debugf("ntlm> Successfully injected NTLM to connection")
	return conn, nil
}
