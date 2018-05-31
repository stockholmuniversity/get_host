package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"gethost/internal"
	"suversion"
)

func main() {
	suversion.PrintVersionAndExit()

	if len(os.Args) != 2 {
		fmt.Println("Need one hostname to look for as argument")
		os.Exit(1)
	}

	tracer, closer := gethost.JaegerInit("gethost-client")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	span := tracer.StartSpan("Get-hosts")
	defer span.Finish()
	ctx := opentracing.ContextWithSpan(context.Background(), span)

	hostToGet := os.Args[1]

	r, err := getFromServer(ctx, hostToGet)
	if err != nil {
		log.Println(err)
	}
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
		go gethost.GetRRforZone(ctx, z, hostToGet, c)
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

func getFromServer(ctx context.Context, z string) ([]string, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "getFromServer")
	defer span.Finish()

	url := "http://localhost:8080/" + z
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

	resp, err := http.DefaultClient.Do(req)
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
