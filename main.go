package main

import (
	"apt_cacher_go/proxy"
	"apt_cacher_go/proxy/certs"
	"flag"
	"log"
)

func main() {
	address := flag.String("listen", "127.0.0.1:9999", "Address and port to listen on")
	caCertFile := flag.String("ca-cert", "ssl/ca.crt", "Path to CA certificate file")
	caKeyFile := flag.String("ca-key", "ssl/ca.key", "Path to CA private key file")
	cacheDir := flag.String("cache-dir", "var/cache/", "Path to cache directory")
	flag.Parse()

	ca, err := certs.NewPrivateCA(*caCertFile, *caKeyFile)
	if err != nil {
		log.Fatalf("Failed to create CA: %v", err)
	}

	proxy, err := proxy.NewCachingMitmProxy(*cacheDir, ca)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	log.Println("Starting proxy server on", *address)
	proxy.Listen(*address)
}
