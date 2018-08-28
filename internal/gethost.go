package gethost

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/miekg/dns"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	config "github.com/uber/jaeger-client-go/config"
)

// GetRRforZone send all CNAME and A records that match 'hostToGet' over channel c.
// If 'hostToGet' is empty all CNAME and A records for zone z will be returned.
// This function is well suited to be started in parallel as an go routine.
func GetRRforZone(ctx context.Context, zone string, hostToGet string, c chan map[string][]dns.RR, verbose bool) {
	span, _ := opentracing.StartSpanFromContext(ctx, "GetRRforZone")
	span.SetTag("zone", zone)
	defer span.Finish()

	t := &dns.Transfer{}
	m := &dns.Msg{}
	m.SetAxfr(zone)
	e, err := t.In(m, "***REMOVED***:53")
	if err != nil {
		fmt.Println("Got error: ", err)
		os.Exit(1)
	}

	dnsRR := map[string][]dns.RR{}
	for envelope := range e { // Range read from channel e
		for _, rr := range envelope.RR { // Iterate over all Resource Records
			name := strings.TrimRight(rr.Header().Name, ".")
			rrtype := rr.Header().Rrtype
			//ttl := rr.Header().Ttl

			if rrtype == dns.TypeA || rrtype == dns.TypeCNAME {
				if hostToGet != "" {
					if strings.Contains(name, hostToGet) {
						tempSlice := dnsRR[name]
						dnsRR[name] = append(tempSlice, rr)
					}
				} else {
					tempSlice := dnsRR[name]
					dnsRR[name] = append(tempSlice, rr)
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
