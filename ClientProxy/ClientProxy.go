// ClientProxy
package main

import (
	"/miekg/dns-master"
	"flag"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

type ClientProxy struct {
	ACCESS      []*net.IPNet
	SERVERS     []string
	s_len       int
	entries     int64
	max_entries int64
	NOW         int64
	giant       *sync.RWMutex
	timeout     time.Duration
}

func _D(fmt string, v ...interface{}) {
	if DEBUG {
		log.Printf(fmt, v...)
	}
}
func (this ClientProxy) refused(w dns.ResponseWriter, req *dns.Msg) {
	m := new(dns.Msg)
	for _, r := range req.Extra {
		if r.Header().Rrtype == dns.TypeOPT {
			m.SetEdns0(4096, r.(*dns.OPT).Do())
		}
	}
	m.SetRcode(req, dns.RcodeRefused)
	w.WriteMsg(m)
}
func (this ClientProxy) ServeDNS(w dns.ResponseWriter, request *dns.Msg) {
	c := new(dns.Client)
	c.ReadTimeout = this.timeout
	c.WriteTimeout = this.timeout
	if response, rtt, err := c.Exchange(request, this.SERVERS[rand.Intn(this.s_len)]); err == nil {
		_D("%s: request took %s", w.RemoteAddr(), rtt)
		w.WriteMsg(response)
	} else {
		this.refused(w, request)
		log.Printf("%s error: %s", w.RemoteAddr(), err)
	}
}

var DEBUG bool

func main() {

	var (
		S_SERVERS       string
		S_LISTEN        string
		S_ACCESS        string
		timeout         int
		max_entries     int64
		expire_interval int64
	)
	flag.StringVar(&S_SERVERS, "proxy", "8.8.8.8:53,8.8.4.4:53", "we proxy requests to those servers")
	flag.StringVar(&S_LISTEN, "listen", "[::]:53", "listen on (both tcp and udp)")
	flag.StringVar(&S_ACCESS, "access", "127.0.0.0/8,10.0.0.0/8", "allow those networks, use 0.0.0.0/0 to allow everything")
	flag.IntVar(&timeout, "timeout", 5, "timeout")
	flag.Int64Var(&expire_interval, "expire_interval", 300, "delete expired entries every N seconds")
	flag.BoolVar(&DEBUG, "debug", false, "enable/disable debug")
	flag.Int64Var(&max_entries, "max_cache_entries", 2000000, "max cache entries")

	flag.Parse()
	servers := strings.Split(S_SERVERS, ",")
	proxyer := ClientProxy{
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
