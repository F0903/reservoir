package main

import (
	"apt_cacher_go/proxy"
	"flag"
	"log"
	"net/http"
)

func main() {
	address := flag.String("listen", "127.0.0.1:9999", "Address and port to listen on")
	caCertFile := flag.String("ca-cert", "cert.pem", "Path to CA certificate file")
	caKeyFile := flag.String("ca-key", "key.pem", "Path to CA private key file")
	cacheDir := flag.String("cache-dir", "cache/", "Path to cache directory")
	flag.Parse()

	proxy, err := proxy.NewCachingMitmProxy(*caCertFile, *caKeyFile, *cacheDir)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	log.Println("Starting proxy server on", *address)
	if err := http.ListenAndServe(*address, proxy); err != nil {
		log.Fatalf("ListenAndServe error: %v", err)
	}
}
