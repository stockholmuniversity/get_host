package main

import (
	"fmt"
	"github.com/miekg/dns"
	//	"log"
	//	"net"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Need one hostname to look for as argument")
		os.Exit(1)
	}
	hostToGet := os.Args[1]
	fmt.Println("Host to get: ", hostToGet)

	t := new(dns.Transfer)
	m := new(dns.Msg)
	m.SetAxfr("***REMOVED***")
	c, err := t.In(m, "***REMOVED***:53")
	if err != nil {
		fmt.Println("Got error: ", err)
		os.Exit(1)
	}
	for envelope := range c { // Range read from channel c
		for _, rr := range envelope.RR { // Iterate over all Resource Records
			name := strings.TrimRight(rr.Header().Name, ".")
			ttl := rr.Header().Ttl

			fmt.Println(name, ttl)
		}
	}
}
