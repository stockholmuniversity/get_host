package main

import (
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

	keys := []string{}
	for k := range dnsRR {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, hostname := range keys {
		if strings.Contains(hostname, hostToGet) {
			fmt.Println("Send match for", hostToGet)
			fmt.Fprintf(w, hostname+"\n")
		}
	}
}
