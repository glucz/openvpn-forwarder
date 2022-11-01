/*
 * Copyright (C) 2019 The "MysteriumNetwork/openvpn-forwarder" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"flag"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"

	"github.com/mysteriumnetwork/openvpn-forwarder/api"
	"github.com/mysteriumnetwork/openvpn-forwarder/proxy"
	"github.com/pkg/errors"
	netproxy "golang.org/x/net/proxy"
)

var proxyAddr = flag.String("proxy.bind", ":8443", "Proxy address for incoming connections")
var proxyAPIAddr = flag.String("proxy.api-bind", ":8000", "HTTP proxy API address")
var proxyUpstreamURL1 = flag.String(
	"proxy.upstream-url1",
	"",
	`Upstream HTTPS proxy where to forward traffic (e.g. "http://superproxy.com:8080")`,
)
var proxyUser1 = flag.String("proxy.user1", "", "HTTP proxy auth user")
var proxyPass1 = flag.String("proxy.pass1", "", "HTTP proxy auth password")
var proxyCountry = flag.String("proxy.country", "", "HTTP proxy country targeting")
var proxyMapPort = FlagArray(
	"proxy.port-map",
	`Explicitly map source port to destination port (separated by comma - "8443:443,18443:8443")`,
)

var stickyStoragePath = flag.String("stickiness-db-path", proxy.MemoryStorage, "Path to the database for stickiness mapping")

var filterHostnames1 = FlagArray(
	"filter.hostnames1",
	`Explicitly forward just several hostnames (separated by comma - "ipinfo.io,ipify.org")`,
)
var filterZones1 = FlagArray(
	"filter.zones1",
	`Explicitly forward just several DNS zones. A zone of "example.com" matches "example.com" and all of its subdomains. (separated by comma - "ipinfo.io,ipify.org")`,
)
var excludeHostnames = FlagArray(
	"exclude.hostnames",
	`Exclude from forwarding several hostnames (separated by comma - "ipinfo.io,ipify.org")`,
)
var excludeZones = FlagArray(
	"exclude.zones",
	`Exclude from forwarding several DNS zones. A zone of "example.com" matches "example.com" and all of its subdomains. (separated by comma - "ipinfo.io,ipify.org")`,
)

////////////////////

var proxyUpstreamURL2 = flag.String(
	"proxy.upstream-url2",
	"",
	`Upstream HTTPS proxy where to forward traffic (e.g. "http://superproxy.com:8080")`,
)
var proxyUser2 = flag.String("proxy.user2", "", "HTTP proxy auth user")
var proxyPass2 = flag.String("proxy.pass2", "", "HTTP proxy auth password")

var filterHostnames2 = FlagArray(
	"filter.hostnames2",
	`Explicitly forward just several hostnames (separated by comma - "ipinfo.io,ipify.org")`,
)
var filterZones2 = FlagArray(
	"filter.zones2",
	`Explicitly forward just several DNS zones. A zone of "example.com" matches "example.com" and all of its subdomains. (separated by comma - "ipinfo.io,ipify.org")`,
)

////////////////////

var proxyUpstreamURL3 = flag.String(
	"proxy.upstream-url3",
	"",
	`Upstream HTTPS proxy where to forward traffic (e.g. "http://superproxy.com:8080")`,
)
var proxyUser3 = flag.String("proxy.user3", "", "HTTP proxy auth user")
var proxyPass3 = flag.String("proxy.pass3", "", "HTTP proxy auth password")

var filterHostnames3 = FlagArray(
	"filter.hostnames3",
	`Explicitly forward just several hostnames (separated by comma - "ipinfo.io,ipify.org")`,
)
var filterZones3 = FlagArray(
	"filter.zones3",
	`Explicitly forward just several DNS zones. A zone of "example.com" matches "example.com" and all of its subdomains. (separated by comma - "ipinfo.io,ipify.org")`,
)

//////////////////////

var enableDomainTracer = flag.Bool("enable-domain-tracer", false, "Enable tracing domain names from requests")

type domainTracker interface {
	Inc(domain string)
	Dump() map[string]uint64
}

func main() {
	flag.Parse()

	dialerUpstreamURL1, err := url.Parse(*proxyUpstreamURL1)
	if err != nil {
		log.Fatalf("Invalid upstream URL: %s", *proxyUpstreamURL1)
	}

	dialerUpstreamURL2, err := url.Parse(*proxyUpstreamURL2)
	if err != nil {
		log.Fatalf("Invalid upstream URL: %s", *proxyUpstreamURL2)
	}

	dialerUpstreamURL3, err := url.Parse(*proxyUpstreamURL3)
	if err != nil {
		log.Fatalf("Invalid upstream URL: %s", *proxyUpstreamURL3)
	}


	sm, err := proxy.NewStickyMapper(*stickyStoragePath)
	if err != nil {
		log.Fatalf("Failed to create sticky mapper, %v", err)
	}

	var domainTracer domainTracker = proxy.NewNoopTracer()
	if *enableDomainTracer {
		domainTracer = proxy.NewDomainTracer()
	}

	apiServer := api.NewServer(*proxyAPIAddr, sm, domainTracer)
	go apiServer.ListenAndServe()

	dialerUpstream3 := proxy.NewDialerHTTPConnect(proxy.DialerDirect, dialerUpstreamURL3.Host, *proxyUser3, *proxyPass3, *proxyCountry)
	dialerUpstream2 := proxy.NewDialerHTTPConnect(proxy.dialerUpstream3, dialerUpstreamURL2.Host, *proxyUser2, *proxyPass2, *proxyCountry)
	dialerUpstream1 := proxy.NewDialerHTTPConnect(proxy.dialerUpstream2, dialerUpstreamURL1.Host, *proxyUser1, *proxyPass1, *proxyCountry)

	var dialer3 netproxy.Dialer
	if len(*filterHostnames3) > 0 || len(*filterZones3) >0 {
		dialerUpstreamFiltered3 := netproxy.NewPerHost(proxy.DialerDirect, dialerUpstream3)
		for _, host := range *filterHostnames3 {
			log.Printf("Redirecting: %s -> %s", host, dialerUpstreamURL3)
			dialerUpstreamFiltered3.AddHost(host)
		}
		for _, zone := range *filterZones3 {
			log.Printf("Redirecting: *.%s -> %s", zone, dialerUpstreamURL3)
			dialerUpstreamFiltered3.AddZone(zone)
		}
		dialer3 = dialerUpstreamFiltered3
	} else {
		dialer3 = dialerUpstream3
		log.Printf("Redirecting: * -> %s", dialerUpstreamURL3)
	}

	var dialer2 netproxy.Dialer
	if len(*filterHostnames2) > 0 || len(*filterZones2) >0 {
		dialerUpstreamFiltered2 := netproxy.NewPerHost(dialer3, dialerUpstream2)
		for _, host := range *filterHostnames2 {
			log.Printf("Redirecting: %s -> %s", host, dialerUpstreamURL2)
			dialerUpstreamFiltered2.AddHost(host)
		}
		for _, zone := range *filterZones2 {
			log.Printf("Redirecting: *.%s -> %s", zone, dialerUpstreamURL2)
			dialerUpstreamFiltered2.AddZone(zone)
		}
		dialer2 = dialerUpstreamFiltered2
	} else {
		dialer2 = dialerUpstream2
		log.Printf("Redirecting: * -> %s", dialerUpstreamURL2)
	}

	var dialer1 netproxy.Dialer
	if len(*filterHostnames1) > 0 || len(*filterZones1) >0 {
		dialerUpstreamFiltered1 := netproxy.NewPerHost(dialer2, dialerUpstream1)
		for _, host := range *filterHostnames1 {
			log.Printf("Redirecting: %s -> %s", host, dialerUpstreamURL1)
			dialerUpstreamFiltered1.AddHost(host)
		}
		for _, zone := range *filterZones1 {
			log.Printf("Redirecting: *.%s -> %s", zone, dialerUpstreamURL1)
			dialerUpstreamFiltered1.AddZone(zone)
		}
		dialer1 = dialerUpstreamFiltered1
	} else {
		dialer1 = dialerUpstream1
		log.Printf("Redirecting: * -> %s", dialerUpstreamURL1)
	}	

	if len(*excludeHostnames) > 0 || len(*excludeZones) > 0 {
		dialerUpstreamExcluded := netproxy.NewPerHost(dialer1, proxy.DialerDirect)
		for _, host := range *excludeHostnames {
			log.Printf("Excluding: %s -> %s", host, dialerUpstreamURL1)
			dialerUpstreamExcluded.AddHost(host)
		}
		for _, zone := range *excludeZones {
			log.Printf("Excluding: *.%s -> %s", zone, dialerUpstreamURL1)
			dialerUpstreamExcluded.AddZone(zone)
		}
		dialer1 = dialerUpstreamExcluded
	}

	portMap, err := parsePortMap(*proxyMapPort, *proxyAddr)
	if err != nil {
		log.Fatal(err)
	}
	proxyServer := proxy.NewServer(dialer1, dialerUpstreamURL1, sm, domainTracer, portMap)

	var wg sync.WaitGroup
	for p := range portMap {
		wg.Add(1)
		go func(p string) {
			log.Print("Serving HTTPS proxy on :", p)
			if err := proxyServer.ListenAndServe(":" + p); err != nil {
				log.Fatalf("Failed to listen http requests: %v", err)
			}
			wg.Done()
		}(p)
	}

	wg.Wait()
}

func parsePortMap(ports flagArray, proxyAddr string) (map[string]string, error) {
	_, port, err := net.SplitHostPort(proxyAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse port")
	}

	portsMap := map[string]string{port: "443"}

	for _, p := range ports {
		portMap := strings.Split(p, ":")
		if len(portMap) != 2 {
			return nil, errors.Errorf("failed to parse port mapping: %s", p)
		}
		portsMap[portMap[0]] = portMap[1]
	}
	return portsMap, nil
}

// FlagArray defines a string array flag
func FlagArray(name string, usage string) *flagArray {
	p := &flagArray{}
	flag.Var(p, name, usage)
	return p
}

type flagArray []string

func (flag *flagArray) String() string {
	return strings.Join(*flag, ",")
}

func (flag *flagArray) Set(s string) error {
	*flag = strings.FieldsFunc(s, func(c rune) bool {
		return c == ','
	})
	return nil
}
