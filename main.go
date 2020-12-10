package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"golang.org/x/net/proxy"
)

func main() {
	fmt.Println("listen :8888")
	ln, err := net.Listen("tcp", "0.0.0.0:8888")

	if err != nil {
		panic(err)
	}

	for {
		conn, e := ln.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				log.Printf("accept temp err: %v", ne)
				continue
			}

			log.Printf("accept err: %v", e)
			return
		}

		go func() {
			log.Println("handling", conn.RemoteAddr().String())
			handle(conn)
			log.Println("handled", conn.RemoteAddr().String())
		}()
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	var sb strings.Builder
	b := make([]byte, 1)
	for {
		_, err := conn.Read(b)

		if err != nil {
			panic(err)
		}
		if string(b) == "\n" {
			break
		}
		sb.WriteString(string(b))
	}
	socksAddress := sb.String()
	log.Println("socksAddress", socksAddress)

	var sb2 strings.Builder
	b2 := make([]byte, 1)
	for {
		_, err := conn.Read(b2)
		if err != nil {
			panic(err)
		}
		if string(b2) == "\n" {
			break
		}
		sb2.WriteString(string(b2))
	}

	upstreamAddress := sb2.String()
	log.Println("upstreamAddress", socksAddress)

	dialer, err := proxy.SOCKS5("tcp", socksAddress, nil, proxy.Direct)
	if err != nil {
		log.Println("proxy create err", err)
		return
	}

	upstream, err := dialer.Dial("tcp", upstreamAddress)
	if err != nil {
		log.Println("dial err", err)
		return
	}
	defer upstream.Close()

	log.Println("upstream RemoteAddr", upstream.RemoteAddr())

	upstreamClosed := make(chan struct{}, 1)
	clientClosed := make(chan struct{}, 1)

	go broker(upstream, conn, clientClosed)
	go broker(conn, upstream, upstreamClosed)

	var waitFor chan struct{}
	select {
	case <-clientClosed:
		log.Println("client closed")
		upstream.Close()
		waitFor = upstreamClosed
	case <-upstreamClosed:
		log.Println("upstream closed")
		conn.Close()
		waitFor = clientClosed
	}

	<-waitFor
}

func broker(dst, src net.Conn, srcClosed chan struct{}) {
	io.Copy(dst, src)

	srcClosed <- struct{}{}
}
