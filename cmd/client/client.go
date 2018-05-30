package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"

	"gethost/internal"
	"suversion"
)

func main() {
	suversion.PrintVersionAndExit()

	if len(os.Args) != 2 {
		fmt.Println("Need one hostname to look for as argument")
		os.Exit(1)
	}
	hostToGet := os.Args[1]

	dnsRR := map[string]uint16{}
	c := make(chan map[string]uint16)

	r, err := getFromServer(hostToGet)
	if err != nil {
		log.Println(err)
	}
	if r != nil {
		for _, i := range r {
			fmt.Println(i)
		}
		os.Exit(0)
	}

	zones := []string{"***REMOVED***", "***REMOVED***", "***REMOVED***", "db.***REMOVED***"}
	for _, z := range zones {
		go gethost.GetRRforZone(z, hostToGet, c)
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

	for _, hostname := range keys {
		fmt.Println(hostname)
	}
}

func getFromServer(z string) ([]string, error) {
	r, err := http.Get("http://localhost:8080/" + z)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	slice := []string{}
	err = json.Unmarshal(body, &slice)
	if err != nil {
		return nil, err
	}

	return slice, nil
}
