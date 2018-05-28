package gethost

import (
	"fmt"
	"github.com/miekg/dns"
	"log"
	"os"
	"strings"
)

// GetRRforZone send all CNAME and A records that match 'hostToGet' over channel c.
// If 'hostToGet' is empty all CNAME and A records for zone z will be returned.
// This function is well suited to be started in parallel as an go routine.
func GetRRforZone(z string, hostToGet string, c chan map[string]uint16) {
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

			if hostToGet != "" {
				if strings.Contains(name, hostToGet) {
					if rrtype == dns.TypeA || rrtype == dns.TypeCNAME {
						dnsRR[name] = rrtype
					}
				}
			} else {
				if rrtype == dns.TypeA || rrtype == dns.TypeCNAME {
					dnsRR[name] = rrtype
				}
			}
		}
	}
	c <- dnsRR
	log.Println("Done writing zone", z)
}
