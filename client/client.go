package main

import (
	"fmt"
	"github.com/miekg/dns"
	"log"
	"net"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Need one hostname to look for as argument")
		os.Exit(1)
	}
	hostToGet := os.Args[1]
	fmt.Println("Host to get: ", hostToGet)

	config, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn("it.su.se"), dns.TypeAXFR)
	m.RecursionDesired = true
	r, _, err := c.Exchange(m, net.JoinHostPort(config.Servers[0], config.Port))
	if r == nil {
		log.Fatalf("*** error: %s\n", err.Error())
	}

	// Stuff must be in the answer section
	for _, a := range r.Answer {
		fmt.Printf("%v\n", a)
	}
}
