package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/chetan/simpleproxy"
)

var (
	bind      = flag.String("bind", "", "Bind hostnames to local ports (app.local.com:7000)")
	httpPort  = flag.Int("http", 80, "Port to listen for HTTP (0 to disable)")
	httpsPort = flag.Int("https", 443, "Port to listen for TLS (HTTPS) (0 to disable)")
)

var message = `<html>
<body>
<h1>502 Bad Gateway</h1>
<p>Can't connect to upstream server, please try again later.</p>
</body>
</html>`

func main() {
	flag.Parse()

	if *bind == "" && len(flag.Args()) == 0 {
		log.Fatal("must specify -bind")
	}

	// create handlers
	vhost := createVhostMux(bind)
	mux := simpleproxy.NewLoggedMux()
	mux.Handle("/", vhost)

	// start listeners
	var wg sync.WaitGroup

	if *httpPort >= 0 {
		wg.Add(1)
		listenAddr := fmt.Sprintf("127.0.0.1:%d", 80)
		fmt.Printf("[*] starting proxy: http://%s\n", listenAddr)
		go func() {
			log.Fatal(http.ListenAndServe(listenAddr, mux))
			wg.Done()
		}()
	}

	if *httpsPort >= 0 {
		wg.Add(1)
		go func() {
			listenTLS(vhost, mux)
			wg.Done()
		}()
	}

	wg.Wait()
}

func listenTLS(vhost *simpleproxy.VhostMux, mux *simpleproxy.LoggedMux) {
	listenAddrTLS := fmt.Sprintf("127.0.0.1:%d", 443)
	fmt.Printf("[*] starting proxy: https://%s\n", listenAddrTLS)
	fmt.Printf("    vhosts:\n")

	server := http.Server{
		Addr:      listenAddrTLS,
		Handler:   mux,
		TLSConfig: createTLSConfig(vhost),
	}
	log.Fatal(server.ListenAndServeTLS("", ""))
}

// Create multi-certificate TLS config from vhost config
func createTLSConfig(vhost *simpleproxy.VhostMux) *tls.Config {
	cfg := &tls.Config{}
	for _, server := range vhost.Servers {
		fmt.Printf("    - %s -> %d\n", server.Host, server.Port)
		cert, err := tls.LoadX509KeyPair(server.Cert, server.Key)
		if err != nil {
			log.Fatal("failed to load keypair:", err)
		}
		cfg.Certificates = append(cfg.Certificates, cert)
	}
	cfg.BuildNameToCertificate()
	return cfg
}

// Create vhost config for each binding
func createVhostMux(bind *string) *simpleproxy.VhostMux {
	servers := make(map[string]*simpleproxy.Vhost)
	bindings := strings.Split(*bind, " ")
	for _, binding := range bindings {
		s := strings.Split(binding, ":")
		hostname := s[0]
		targetPort, err := strconv.Atoi(s[1])
		if err != nil {
			log.Fatal("failed to parse target port:", err)
			os.Exit(1)
		}

		proxy := simpleproxy.CreateProxy(url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", targetPort)})

		vhost := &simpleproxy.Vhost{
			Host: hostname, Port: targetPort, Handler: proxy,
		}

		if *useTLS {
			vhost.Cert, vhost.Key, err = simpleproxy.MakeCert(hostname)
			if err != nil {
				log.Fatalf("failed to generate cert for host %s", hostname)
			}
		}

		servers[hostname] = vhost
	}

	return &simpleproxy.VhostMux{Servers: servers}
}
