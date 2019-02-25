package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/miekg/dns"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/stockholmuniversity/goversionflag"

	gethost "gethost/internal"
)

type cache struct {
	data map[string][]dns.RR
	soas []dns.SOA
	sync.RWMutex
	age time.Time
}

func (c cache) Age() time.Duration {
	t := time.Since(c.age)
	return t.Truncate(time.Second)
}

func (c *cache) Len() int {
	c.RLock()
	n := len(c.data)
	c.RUnlock()
	return n
}

var dnsRR cache
var tracer opentracing.Tracer
var verbose *bool
var startTime time.Time

func init() {
	startTime = time.Now()
}

func main() {
	verbose = flag.Bool("verbose", false, "Print reload and responses to questions to standard out")
	configFile := flag.String("configfile", "", "Configuation file")
	goversionflag.PrintVersionAndExit()

	if *configFile == "" {
		log.Fatalln("Need configuration file.")
	}

	config, err := gethost.NewConfig(configFile)
	if err != nil {
		log.Println("Got error when parsing configuration file: " + err.Error())
		os.Exit(1)
	}

	var closer io.Closer

	if *verbose == true {
		config.Verbose = true
	}

	if config.Tracing == true {
		tracer, closer = gethost.JaegerInit("gethost-server")
		defer closer.Close()
	} else {
		tracer = opentracing.GlobalTracer()
	}
	opentracing.SetGlobalTracer(tracer)

	go schedUpdate(tracer, config)
	handleRequests(config)
}

func schedUpdate(tracer opentracing.Tracer, config *gethost.Config) {
	log.Printf("Starting scheduled update of cache every %v seconds.\n", config.TTL)
	for {
		if config.Verbose == true {
			log.Println("Scheduled update in progress.")
		}
		span := tracer.StartSpan("schedUpdate")
		ctx := opentracing.ContextWithSpan(context.Background(), span)

		updateDNS(ctx, config)
		span.Finish()
		time.Sleep(time.Duration(config.TTL) * time.Second)
	}
}

func updateDNS(ctx context.Context, config *gethost.Config) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "updateDNS")
	defer span.Finish()

	dnsRRdataNew, soas, err := buildDNS(ctx, config)
	if err != nil {
		log.Printf("Could not build DNS; %s", err)
		return
	}
	dnsRR.Lock()
	dnsRR.data = dnsRRdataNew
	dnsRR.age = time.Now()
	dnsRR.soas = soas
	dnsRR.Unlock()

}

func buildDNS(ctx context.Context, config *gethost.Config) (map[string][]dns.RR, []dns.SOA, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "buildDNS")
	defer span.Finish()
	var (
		gotErr []error
		soas   []dns.SOA
	)

	zones := gethost.Zones(config)
	c := make(chan gethost.GetRRforZoneResult)
	defer close(c)

	for _, s := range zones {
		z := s.Header().Name
		go gethost.GetRRforZone(ctx, z, "", c, config)
	}

	dnsRRdataNew := map[string][]dns.RR{}
	for range zones {
		m := <-c
		if m.Err != nil {
			gotErr = append(gotErr, m.Err)
		} else {
			soas = append(soas, *m.SOA.SOA)
			for k, v := range m.SOA.RR {
				dnsRRdataNew[k] = v
			}
		}
	}
	if gotErr != nil {
		var ret string
		for _, v := range gotErr {
			ret = ret + " " + v.Error()
		}
		return nil, nil, errors.New("Could not build cache, at least one error: " + ret)
	}
	return dnsRRdataNew, soas, nil
}

func handleRequests(config *gethost.Config) {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/hosts/{id}", wrapper(config, httpResponse))
	myRouter.HandleFunc("/hosts/{id}/{nc}", wrapper(config, httpResponse))
	myRouter.HandleFunc("/version", httpVersion)
	myRouter.HandleFunc("/status", wrapper(config, httpStatus))
	addr := ":" + strconv.Itoa(config.ServerPort)
	log.Println("Staring server on", addr)
	log.Fatal(http.ListenAndServe(addr, myRouter))
}

func wrapper(config *gethost.Config, handler func(w http.ResponseWriter, r *http.Request, config *gethost.Config)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, config)
	}
}

func httpResponse(w http.ResponseWriter, r *http.Request, config *gethost.Config) {
	spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	span := tracer.StartSpan("httpResponse", ext.RPCServerOption(spanCtx))
	ctx := opentracing.ContextWithSpan(context.Background(), span)
	defer span.Finish()

	vars := mux.Vars(r)
	hostToGet := vars["id"]
	noCache := vars["nc"]

	if noCache == "nc" {
		log.Println("got nc flag")
		updateDNS(ctx, config)
	}

	hostnames := []string{}

	dnsRR.RLock()
	for hostname := range dnsRR.data { // TODO, method on cache struct?
		if strings.Contains(hostname, hostToGet) {
			hostnames = append(hostnames, hostname)
		}
	}
	dnsRR.RUnlock()
	sort.Strings(hostnames)

	j, err := json.Marshal(hostnames)
	if err != nil {
		log.Println("Error:", err)
		os.Exit(1)
	}

	if config.Verbose == true {
		log.Println("Send match for " + hostToGet + ": " + string(j))
	}
	fmt.Fprintf(w, string(j))
}

func httpVersion(w http.ResponseWriter, r *http.Request) {
	buildversion := goversionflag.GetBuildInformation()
	buildSlice := []string{}
	missingBuildInfo := false
	for k, v := range buildversion {
		buildSlice = append(buildSlice, k+": "+v+"\n")
		if v == "" {
			missingBuildInfo = true
		}
	}
	sort.Strings(buildSlice)
	for _, v := range buildSlice {
		fmt.Fprintf(w, v)
	}
	if missingBuildInfo {
		fmt.Fprintf(w, `Do not have complete buildinfo, see documentaion:
https://github.com/stockholmuniversity/goversionflag
https://godoc.org/github.com/stockholmuniversity/goversionflag
`)
	}

}

func httpStatus(w http.ResponseWriter, r *http.Request, config *gethost.Config) {
	spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	span := tracer.StartSpan("httpStatus", ext.RPCServerOption(spanCtx))
	defer span.Finish()

	dnsRR.RLock()
	fmt.Fprintf(w, "Zones cached:\n")
	for _, s := range dnsRR.soas {
		z := s.Header().Name
		serial := s.Serial
		fmt.Fprintf(w, "%s serial: %d\n", z, serial)
	}

	fmt.Fprintf(w, "Cache size: %d\n", dnsRR.Len())
	fmt.Fprintf(w, "Cache age: %s\n", dnsRR.Age())
	fmt.Fprintf(w, "Uptime: %s\n", uptime())
	dnsRR.RUnlock()

}

func uptime() time.Duration {
	t := time.Since(startTime)
	return t.Truncate(time.Second)
}
