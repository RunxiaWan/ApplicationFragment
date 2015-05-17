// ClientProxy
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
var DEBUG bool = false

// called for debug output
func _D(fmt string, v ...interface{}) {
	if DEBUG {
		log.Printf(fmt, v...)
	}
}

// this structure will be used by the dns.ListenAndServe() method
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

// SRVFAIL result for serious problems
func (this ClientProxy) SRVFAIL(w dns.ResponseWriter, req *dns.Msg) {
	m := new(dns.Msg)
	m.SetRcode(req, dns.RcodeServerFailure)
	w.WriteMsg(m)
}

func (this ClientProxy) ServeDNS(w dns.ResponseWriter, request *dns.Msg) {
	// if we don't have EDNS0 in the packet, add it now
	// TODO: in principle we should check packet size here, since we have made it bigger,
	//       but for this demo code we will just rely on most queries being really small
	proxy_req := *request
	opt := proxy_req.IsEdns0()
	var client_buf_size uint16
	if opt == nil {
		proxy_req.SetEdns0(512, false)
		client_buf_size = 512
		_D("%s QID:%d adding EDNS0 to packet", w.RemoteAddr(), request.Id)
		opt = proxy_req.IsEdns0()
	} else {
		client_buf_size = opt.UDPSize()
	}

	// add our custom EDNS0 option
	local_opt := new(dns.EDNS0_LOCAL)
	local_opt.Code = dns.EDNS0LOCALSTART
	opt.Option = append(opt.Option, local_opt)

	// create a connection to the server
	// XXX: for now we will only handle UDP - this will break in unpredictable ways in production!
	conn, err := dns.DialTimeout("udp", this.SERVERS[rand.Intn(len(this.SERVERS))], this.timeout)
	if err != nil {
		_D("%s QID:%d error setting up UDP socket: %s", w.RemoteAddr(), request.Id, err)
		this.SRVFAIL(w, request)
		return
	}
	defer conn.Close()

	// set our timeouts
	// TODO: we need to insure that our timeouts work like we expect
	conn.SetReadDeadline(time.Now().Add(this.timeout))
	conn.SetWriteDeadline(time.Now().Add(this.timeout))

	// send our query
	err = conn.WriteMsg(&proxy_req)
	if err != nil {
		_D("%s QID:%d error writing message", w.RemoteAddr(), request.Id)
		this.SRVFAIL(w, request)
		return
	}

	// wait for our reply
	for {
		// TODO: verify that we are checking source/dest ports in conn.ReadMsg()
		response, err := conn.ReadMsg()
		// some sort of error reading reply
		if err != nil {
			_D("%s QID:%d error reading message: %s", w.RemoteAddr(), request.Id, err)
			this.SRVFAIL(w, request)
			return
		}
		// got a response, life is good
		if response.Id == request.Id {
			_D("%s QID:%d got reply", w.RemoteAddr(), request.Id)
			w.WriteMsg(response)
			break
		}
		// got a response, but it was for a different QID... ignore
		_D("%s QID:%d ignoring reply to wrong QID:%d", w.RemoteAddr(), request.Id, response.Id)
	}
	client_buf_size = client_buf_size // XXX: get rid of unused variable warning...
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
