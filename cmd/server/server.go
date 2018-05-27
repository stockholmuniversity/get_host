package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"

	"gethost"
	"github.com/gorilla/mux"
	"suversion"
)

func main() {
	suversion.PrintVersionAndExit()

	//dnsRR := map[string]uint16{}
	dnsRR := buildAndUpdateDNS()

	keys := []string{}
	for k := range dnsRR {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, hostname := range keys {
		fmt.Println(hostname)
	}
	handleRequests()
}

func buildAndUpdateDNS() map[string]uint16 {
	zones := []string{"***REMOVED***", "***REMOVED***", "***REMOVED***", "db.***REMOVED***"}
	dnsRR := map[string]uint16{}
	c := make(chan map[string]uint16)

	for _, z := range zones {
		go gethost.GetRRforZone(z, "", c)
	}

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
	key := vars["id"]

	fmt.Fprintf(w, "Key: "+key)
}
