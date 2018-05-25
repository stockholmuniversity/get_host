package main

import (
	"fmt"
	"github.com/miekg/dns"
	"log"
	"os"
	"sort"
	"strings"
	"suversion"
)

func main() {
	suversion.PrintVersionAndExit()
	if len(os.Args) != 2 {
		fmt.Println("Need one hostname to look for as argument")
		os.Exit(1)
	}
	hostToGet := os.Args[1]

	dnsRR := map[string]uint16{}
	c := make(chan map[string]uint16)
	zones := []string{"***REMOVED***", "***REMOVED***", "***REMOVED***", "db.***REMOVED***"}
	for _, z := range zones {
		go getRRforZone(z, hostToGet, c)
	}

	for range zones {
		m := <-c
		for k, v := range m {
			dnsRR[k] = v
		}
	}

	keys := []string{}
	for k := range dnsRR {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, hostname := range keys {
		fmt.Println(hostname)
	}
}

func getRRforZone(z string, hostToGet string, c chan map[string]uint16) {
	t := &dns.Transfer{}
	m := &dns.Msg{}
	m.SetAxfr(z)
	e, err := t.In(m, "***REMOVED***:53")
	if err != nil {
		fmt.Println("Got error: ", err)
		os.Exit(1)
	}

	dnsRR := map[string]uint16{}
	for envelope := range e { // Range read from channel e
		for _, rr := range envelope.RR { // Iterate over all Resource Records
			name := strings.TrimRight(rr.Header().Name, ".")
			rrtype := rr.Header().Rrtype
			//ttl := rr.Header().Ttl

			if strings.Contains(name, hostToGet) {
				if rrtype == dns.TypeA || rrtype == dns.TypeCNAME {
					dnsRR[name] = rrtype
				}
			}
		}
	}
	c <- dnsRR
	log.Println("Done writing zone ", z)
}
