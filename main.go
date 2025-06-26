package main

import (
	"flag"
	"log"
	"mitm_proxy/proxy"
	"net/http"
)

func main() {
	address := flag.String("address", "127.0.0.1:9999", "Address to listen on")
	caCertFile := flag.String("ca-cert", "cert.pem", "Path to CA certificate file")
	caKeyFile := flag.String("ca-key", "key.pem", "Path to CA private key file")
	flag.Parse()

	proxy, err := proxy.NewMitmProxy(*caCertFile, *caKeyFile)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	log.Println("Starting proxy server on", *address)
	if err := http.ListenAndServe(*address, proxy); err != nil {
		log.Fatalf("ListenAndServe error: %v", err)
	}
}
