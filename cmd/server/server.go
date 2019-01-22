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

var dnsRR map[string][]dns.RR
var mtx sync.RWMutex
var tracer opentracing.Tracer
var verbose *bool

func main() {
	useTracing := flag.Bool("tracing", false, "Enable tracing of calls.")
	port := flag.Int("port", 8080, "Port for server")
	timeout := flag.Int("ttl", 900, "Cache reload interval in seconds")
	verbose = flag.Bool("verbose", false, "Print reload and responses to questions to standard out")
	goversionflag.PrintVersionAndExit()

	var closer io.Closer

	if *useTracing == true {
		tracer, closer = gethost.JaegerInit("gethost-server")
		defer closer.Close()
	} else {
		tracer = opentracing.GlobalTracer()
	}
	opentracing.SetGlobalTracer(tracer)

	go schedUpdate(tracer, *timeout)
	handleRequests(*port)
}

func printVerbose(output string) {
	if *verbose == true {
		log.Println(output)
	}
}

func schedUpdate(tracer opentracing.Tracer, timeout int) {
	log.Printf("Starting scheduled update of cache every %v seconds.\n", timeout)
	for {
		printVerbose("Scheduled update in progress.")
		span := tracer.StartSpan("schedUpdate")
		ctx := opentracing.ContextWithSpan(context.Background(), span)

		updateDNS(ctx)
		span.Finish()
		time.Sleep(time.Duration(timeout) * time.Second)
	}
}

func updateDNS(ctx context.Context) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "updateDNS")
	defer span.Finish()

	dnsRRnew, err := buildDNS(ctx, *verbose)
	if err != nil {
		log.Printf("Could not build DNS; %s", err)
		return
	}
	mtx.Lock()
	dnsRR = dnsRRnew
	mtx.Unlock()

}

func buildDNS(ctx context.Context, verbose bool) (map[string][]dns.RR, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "buildDNS")
	defer span.Finish()
	var gotErr error

	zones := gethost.Zones()
	c := make(chan gethost.GetRRforZoneResult)

	for _, s := range zones {
		z := s.Header().Name
		go gethost.GetRRforZone(ctx, z, "", c, verbose)
	}

	dnsRRnew := map[string][]dns.RR{}
	for range zones {
		m := <-c
		if m.Err != nil {
			gotErr = m.Err
		}
		for k, v := range m.SOA.RR {
			dnsRRnew[k] = v
		}
	}
	if gotErr != nil {
		return nil, errors.New("Could not build cache, at least one error: ")
	}
	return dnsRRnew, nil
}

func handleRequests(port int) {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/{id}", httpResponse)
	myRouter.HandleFunc("/{id}/{nc}", httpResponse)
	addr := ":" + strconv.Itoa(port)
	log.Println("Staring server on", addr)
	log.Fatal(http.ListenAndServe(addr, myRouter))
}

func httpResponse(w http.ResponseWriter, r *http.Request) {
	spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	span := tracer.StartSpan("httpResponse", ext.RPCServerOption(spanCtx))
	ctx := opentracing.ContextWithSpan(context.Background(), span)
	defer span.Finish()

	vars := mux.Vars(r)
	hostToGet := vars["id"]
	noCache := vars["nc"]

	if noCache == "nc" {
		log.Println("got nc flag")
		updateDNS(ctx)
	}

	hostnames := []string{}

	mtx.RLock()
	for hostname := range dnsRR {
		if strings.Contains(hostname, hostToGet) {
			hostnames = append(hostnames, hostname)
		}
	}
	mtx.RUnlock()
	sort.Strings(hostnames)

	j, err := json.Marshal(hostnames)
	if err != nil {
		log.Println("Error:", err)
		os.Exit(1)
	}

	printVerbose("Send match for " + hostToGet + ": " + string(j))
	fmt.Fprintf(w, string(j))
}
