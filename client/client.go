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
	var aRecords map[string]int
	aRecords = make(map[string]int)
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
			rrtype := rr.Header().Rrtype
			//ttl := rr.Header().Ttl

			if rrtype == dns.TypeA {
				if strings.Contains(name, hostToGet) {
					aRecords[name] = 1
				}
			}
		}
	}
	for hostname := range aRecords {
		fmt.Println(hostname)
	}
}
