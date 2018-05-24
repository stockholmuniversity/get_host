package main

import (
	"fmt"
	"github.com/miekg/dns"
	"os"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Need one hostname to look for as argument")
		os.Exit(1)
	}
	hostToGet := os.Args[1]
	dnsRR := map[string]uint16{}
	//	t := new(dns.Transfer)
	//	m := new(dns.Msg)
	for _, z := range []string{"***REMOVED***", "***REMOVED***", "db.***REMOVED***", "***REMOVED***"} {
		t := &dns.Transfer{}
		m := &dns.Msg{}
		m.SetAxfr(z)
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

				if strings.Contains(name, hostToGet) {
					if rrtype == dns.TypeA || rrtype == dns.TypeCNAME {
						dnsRR[name] = rrtype
					}
				}
			}
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
