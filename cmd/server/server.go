package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"gethost/internal"
	"suversion"
)

var dnsRR map[string]uint16

func main() {
	suversion.PrintVersionAndExit()

	go updateDNS()
	handleRequests()
}

func updateDNS() {
	for {
        // TODO Dont just replace global dnsRR or use mutex
		dnsRR = buildAndUpdateDNS()
		time.Sleep(10 * time.Second)
	}
}

func buildAndUpdateDNS() map[string]uint16 {
	zones := []string{"***REMOVED***", "***REMOVED***", "***REMOVED***", "db.***REMOVED***"}
	c := make(chan map[string]uint16)

	for _, z := range zones {
		go gethost.GetRRforZone(z, "", c)
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
	vars := mux.Vars(r)
	hostToGet := vars["id"]

	hostnames := []string{}
    // TODO use mutex read-lock
	for hostname := range dnsRR {
		if strings.Contains(hostname, hostToGet) {
			hostnames = append(hostnames, hostname)
		}
	}
	sort.Strings(hostnames)

	j, err := json.Marshal(hostnames)
	if err != nil {
		log.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("Send match for", hostToGet, ":", string(j))
	fmt.Fprintf(w, string(j))
}
