package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Encinarus/genconplanner/internal/background"
	"github.com/Encinarus/genconplanner/internal/postgres"
)

var sourceFile = flag.String("eventFile", "http://www.gencon.com/downloads/events.xlsx", "file path or url to load from")
var overrideDns = flag.Bool("overrideDNS", false, "Override DNS settings (useful for docker)")

func setGoogleDns() {
	var (
		dnsResolverIP        = "8.8.8.8:53" // Google DNS resolver.
		dnsResolverProto     = "udp"        // Protocol to use for the DNS resolver
		dnsResolverTimeoutMs = 5000         // Timeout (ms) for the DNS resolver (optional)
	)

	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Duration(dnsResolverTimeoutMs) * time.Millisecond,
				}
				return d.DialContext(ctx, dnsResolverProto, dnsResolverIP)
			},
		},
	}

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}

	http.DefaultTransport.(*http.Transport).DialContext = dialContext

}

func main() {
	flag.Parse()

	db, err := postgres.OpenDb()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if len(*sourceFile) == 0 {
		log.Fatalf("You must specify a source file")
	}

	if *overrideDns {
		setGoogleDns()
	}

	background.UpdateEventsFromGencon(db, *sourceFile)
}
