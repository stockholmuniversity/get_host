package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"gethost/internal"
	"suversion"
)

var dnsRR map[string]uint16
var mtx sync.RWMutex

func main() {
	suversion.PrintVersionAndExit()

	go updateDNS()
	handleRequests()
}

func updateDNS() {
	for {
		dnsRRnew := buildAndUpdateDNS()
		mtx.Lock()
		dnsRR = dnsRRnew
		mtx.Unlock()
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
