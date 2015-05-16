// ServerProxy project main.go
package main

import (
	"github.com/miekg/dns"
	"flag"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

// flag whether we want to emit debug output
var DEBUG bool := false

// called for debug output
func _D(fmt string, v ...interface{}) {
	if DEBUG {
		log.Printf(fmt, v...)
	}
}

// this structure will be used the dns.ListenAndServe() method
type ServerProxy struct {
	ACCESS      []*net.IPNet
	SERVERS     []string
	s_len       int
	entries     int64
	max_entries int64
	NOW         int64
	giant       *sync.RWMutex
	timeout     time.Duration
}

func (this ServerProxy) refused(w dns.ResponseWriter, req *dns.Msg) {
	m := new(dns.Msg)
	for _, r := range req.Extra {
		if r.Header().Rrtype == dns.TypeOPT {
			m.SetEdns0(4096, r.(*dns.OPT).Do())
		}
	}
	m.SetRcode(req, dns.RcodeRefused)
	w.WriteMsg(m)
}

// our ServeDNS interface, which gets invoked on every DNS message
func (this ServerProxy) ServeDNS(w dns.ResponseWriter, request *dns.Msg) {
	c := new(dns.Client)
	c.ReadTimeout = this.timeout
	c.WriteTimeout = this.timeout
	if response, rtt, err := c.Exchange(request, this.SERVERS[rand.Intn(this.s_len)]); err == nil {
		_D("%s: request took %s", w.RemoteAddr(), rtt)
		w.WriteMsg(response)
	} else {
		// TODO: we should be careful and return the correct error from the server
		this.refused(w, request)
		log.Printf("%s error: %s", w.RemoteAddr(), err)
	}
}

func main() {

	var (
		S_SERVERS       string
		S_LISTEN        string
		S_ACCESS        string
		timeout         int
		max_entries     int64
		expire_interval int64
	)
	flag.StringVar(&S_SERVERS, "proxy", "127.0.0.1:53", "we proxy requests to those servers")
	flag.StringVar(&S_LISTEN, "listen", "[::]:8000", "listen on (both tcp and udp)")
	flag.StringVar(&S_ACCESS, "access", "0.0.0.0/0", "allow those networks, use 0.0.0.0/0 to allow everything")
	flag.IntVar(&timeout, "timeout", 5, "timeout")
	flag.Int64Var(&expire_interval, "expire_interval", 300, "delete expired entries every N seconds")
	flag.BoolVar(&DEBUG, "debug", false, "enable/disable debug")
	flag.Int64Var(&max_entries, "max_cache_entries", 2000000, "max cache entries")

	flag.Parse()
	servers := strings.Split(S_SERVERS, ",")
	proxyer := ServerProxy{
		giant:       new(sync.RWMutex),
		ACCESS:      make([]*net.IPNet, 0),
		SERVERS:     servers,
		s_len:       len(servers),
		NOW:         time.Now().UTC().Unix(),
		entries:     0,
		timeout:     time.Duration(timeout) * time.Second,
		max_entries: max_entries}

	for _, mask := range strings.Split(S_ACCESS, ",") {
		_, cidr, err := net.ParseCIDR(mask)
		if err != nil {
			panic(err)
		}
		_D("added access for %s\n", mask)
		proxyer.ACCESS = append(proxyer.ACCESS, cidr)
	}
	for _, addr := range strings.Split(S_LISTEN, ",") {
		_D("listening @ %s\n", addr)
		go func() {
			if err := dns.ListenAndServe(addr, "udp", proxyer); err != nil {
				log.Fatal(err)
			}
		}()

		go func() {
			if err := dns.ListenAndServe(addr, "tcp", proxyer); err != nil {
				log.Fatal(err)
			}
		}()
	}

	for {
		proxyer.NOW = time.Now().UTC().Unix()
		time.Sleep(time.Duration(1) * time.Second)
	}
}
