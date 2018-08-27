package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"gethost/internal"
	"suGoVersion"
)

func main() {

	useTracing := flag.Bool("tracing", false, "Enable tracing of calls.")
	useNC := flag.Bool("nc", false, "No Cache. Force reload of cache")
	getAllHosts := flag.Bool("a", false, "Get all hosts")
	suGoVersion.PrintVersionAndExit()

	var hostToGet string
	flagsLeftover := flag.Args()
	if len(flagsLeftover) > 0 {
		hostToGet = flagsLeftover[0]
	}

	if hostToGet == "" && *getAllHosts == false {
		log.Println("Need part of hostname to match against")
		os.Exit(1)
	}

	var tracer opentracing.Tracer
	var closer io.Closer

	if *useTracing == true {
		tracer, closer = gethost.JaegerInit("gethost-client")
		defer closer.Close()
	} else {
		tracer = opentracing.GlobalTracer()
	}
	span := tracer.StartSpan("Get-hosts")
	defer span.Finish()
	ctx := opentracing.ContextWithSpan(context.Background(), span)

	// Server uses hostname/nc to force reload of cache.
	if *useNC == true {
		hostToGet = hostToGet + "/nc"
	}
	r, err := getFromServer(ctx, hostToGet)
	if err != nil {
		log.Println(err)
	}
	// No match from server, do lookup ourself
	if r == nil {
		r = getFromDNS(ctx, hostToGet)
	}

	for _, i := range r {
		fmt.Println(i)
	}

}

func getFromDNS(ctx context.Context, hostToGet string) []string {
	span, ctx := opentracing.StartSpanFromContext(ctx, "getFromDNS")
	defer span.Finish()

	dnsRR := map[string]uint16{}
	c := make(chan map[string]uint16)

	zones := []string{"***REMOVED***", "***REMOVED***", "***REMOVED***", "db.***REMOVED***"}
	for _, z := range zones {
		go gethost.GetRRforZone(ctx, z, hostToGet, c, false)
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
	return keys

}

func getFromServer(ctx context.Context, hostToGet string) ([]string, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "getFromServer")
	defer span.Finish()

	url := "http://localhost:8080/" + hostToGet
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err.Error())
	}

	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, url)
	ext.HTTPMethod.Set(span, "GET")
	span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)

	client := &http.Client{Timeout: 2000 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	slice := []string{}
	err = json.Unmarshal(body, &slice)
	if err != nil {
		return nil, err
	}

	return slice, nil
}
