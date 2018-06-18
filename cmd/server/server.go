package main

import (
	"context"
	"encoding/json"
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
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"gethost/internal"
	"suversion"
)

var dnsRR map[string]uint16
var mtx sync.RWMutex
var tracer opentracing.Tracer

func main() {
	useTracing := flag.Bool("tracing", false, "Enable tracing of calls.")
	port := flag.Int("port", 8080, "Port for server")
	timeout := flag.Int("ttl", 900, "Cache reload interval in seconds")
	suversion.PrintVersionAndExit()

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

func schedUpdate(tracer opentracing.Tracer, timeout int) {
	for {
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

	dnsRRnew := buildDNS(ctx)
	mtx.Lock()
	dnsRR = dnsRRnew
	mtx.Unlock()

	span.Finish()
}

func buildDNS(ctx context.Context) map[string]uint16 {
	span, ctx := opentracing.StartSpanFromContext(ctx, "buildDNS")
	defer span.Finish()

	zones := []string{"***REMOVED***", "***REMOVED***", "***REMOVED***", "db.***REMOVED***"}
	c := make(chan map[string]uint16)

	for _, z := range zones {
		go gethost.GetRRforZone(ctx, z, "", c, true)
	}

	dnsRR := map[string]uint16{}
	for range zones {
		m := <-c
		for k, v := range m {
			dnsRR[k] = v
		}
	}
	return dnsRR
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

	fmt.Println("Send match for", hostToGet, ":", string(j))
	fmt.Fprintf(w, string(j))
}
