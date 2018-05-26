package gethost

import (
	"fmt"
	"gethost"
	"os"
	"sort"
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
