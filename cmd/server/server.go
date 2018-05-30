package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
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
	suversion.PrintVersionAndExit()

	var closer io.Closer
	tracer, closer = gethost.JaegerInit("gethost-server")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	go updateDNS(tracer)
	handleRequests()
}

func updateDNS(tracer opentracing.Tracer) {

	for {
		span := tracer.StartSpan("updateDNS")
		ctx := opentracing.ContextWithSpan(context.Background(), span)

		dnsRRnew := buildAndUpdateDNS(ctx)
		mtx.Lock()
		dnsRR = dnsRRnew
		mtx.Unlock()

		span.Finish()
		time.Sleep(10 * time.Second)
	}
}

func buildAndUpdateDNS(ctx context.Context) map[string]uint16 {
	zones := []string{"***REMOVED***", "***REMOVED***", "***REMOVED***", "db.***REMOVED***"}
	c := make(chan map[string]uint16)

	for _, z := range zones {
		go gethost.GetRRforZone(ctx, z, "", c)
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

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/{id}", httpResponse)
	port := ":" + os.Args[1]
	log.Fatal(http.ListenAndServe(port, myRouter))
}

func httpResponse(w http.ResponseWriter, r *http.Request) {
	spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
	span := tracer.StartSpan("httpResponse", ext.RPCServerOption(spanCtx))
	defer span.Finish()

	vars := mux.Vars(r)
	hostToGet := vars["id"]

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
