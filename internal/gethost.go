package gethost

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/miekg/dns"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	config "github.com/uber/jaeger-client-go/config"
)

// SOAwithRR is an data structure for slected dns.RR and corresponding SOA
type SOAwithRR struct {
	SOA *dns.SOA
	RR  map[string][]dns.RR
}

// Zones returns a pointer to an slice with dns.SOA RR type for the zones to get AXFR from.
func Zones() []dns.SOA {
	var soas = []dns.SOA{}
	zones := []string{"***REMOVED***", "***REMOVED***", "***REMOVED***", "db.***REMOVED***"}
	for _, z := range zones {
		soa := dns.SOA{}
		soa.Header().Name = z
		soas = append(soas, soa)
	}
	return soas
}

// GetRRforZone send all CNAME and A records that match 'hostToGet' over channel c.
// If 'hostToGet' is empty all CNAME and A records for zone z will be returned.
// This function is well suited to be started in parallel as an go routine.
func GetRRforZone(ctx context.Context, zone string, hostToGet string, c chan SOAwithRR, verbose bool) {
	span, _ := opentracing.StartSpanFromContext(ctx, "GetRRforZone")
	span.SetTag("zone", zone)
	defer span.Finish()

	t := &dns.Transfer{}
	m := &dns.Msg{}
	m.SetAxfr(zone)
	// TODO DNS query to get NS for su.se
	e, err := t.In(m, "***REMOVED***:53")
	if err != nil {
		log.Println("Got error: ", err)
	}

	dnsRR := SOAwithRR{}
	dnsRR.RR = make(map[string][]dns.RR)
	for envelope := range e { // Range read from channel e
		for _, rr := range envelope.RR { // Iterate over all Resource Records
			name := strings.TrimRight(rr.Header().Name, ".")
			rrtype := rr.Header().Rrtype

			if rrtype == dns.TypeSOA {
				dnsRR.SOA = rr.(*dns.SOA)
			}

			if rrtype == dns.TypeA || rrtype == dns.TypeCNAME { // TODO shoud we save AAAA records also?
				if hostToGet != "" {
					if strings.Contains(name, hostToGet) {
						tempSlice := dnsRR.RR[name]
						dnsRR.RR[name] = append(tempSlice, rr)
					}
				} else {
					tempSlice := dnsRR.RR[name]
					dnsRR.RR[name] = append(tempSlice, rr)
				}
			}
		}
	}
	c <- dnsRR
	if verbose == true {
		log.Println("Done writing zone", zone)
	}
}

// JaegerInit initialises jaeger object.
func JaegerInit(service string) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}
	tracer, closer, err := cfg.New(service, config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}
