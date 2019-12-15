package rsdns

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/miekg/dns"

	"github.com/spf13/viper"
)

type data_t map[string]struct {
	IP  string `json:"ip,omitempty"`
	TTL uint32 `json:"ttl,omitempty"`
	Key string `json:"key,omitempty"`
}

var data data_t

func Serve() {
	readData()

	go serveDns()
	go serveHttp()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	fmt.Printf("Signal (%s) received, stopping\n", s)
}

func readData() {
	fn := viper.GetString("data")
	// if !filepath.IsAbs(fn) {
	// 	wd, _ := os.Getwd()
	// 	fn = fmt.Sprintf("%s/%s", wd, fn)
	// 	fn, _ = filepath.Abs(fn)
	// }
	d, err := ioutil.ReadFile(fn)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(d, &data)
	if err != nil {
		panic(err)
	}
}

func writeData() {
	fn := viper.GetString("data")
	d, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(fn, d, 0644)
	if err != nil {
		panic(err)
	}
}

func serveDns() {
	zone := viper.GetString("dns.zone")
	addr := viper.GetString("dns.addr")

	fmt.Printf("RSDNS dns serve on %s (%s)\n", addr, zone)

	dns.HandleFunc(zone, handleDnsQuery)
	server := &dns.Server{Addr: addr, Net: "udp"}
	if err := server.ListenAndServe(); err != nil {
		panic(fmt.Errorf("Failed to setup the dns server: %w\n", err))
	}
}

func serveHttp() {
	addr := viper.GetString("http.addr")
	base := viper.GetString("http.base")

	fmt.Printf("RSDNS http serve on %s (%s)\n", addr, base)

	http.HandleFunc(fmt.Sprintf("%s/plain.php", base), handleHttpPlain)
	if err := http.ListenAndServe(addr, nil); err != nil {
		panic(fmt.Errorf("Failed to setup the http server: %w\n", err))
	}
}

func handleDnsQuery(w dns.ResponseWriter, r *dns.Msg) {
	m := &dns.Msg{}
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
		_ = w.WriteMsg(m)
	}()

	m.SetReply(r)

	q := r.Question[0]

	zone := viper.GetString("dns.zone")
	lenth := len(q.Name) - len(zone) - 1
	if lenth <= 0 {
		return
	}
	host := q.Name[0:lenth]
	item, ok := data[host]
	if !ok {
		return
	}
	a := net.ParseIP(item.IP)
	if a == nil {
		return
	}
	ttl := item.TTL
	if ttl == 0 {
		ttl = 5
	}

	answer := []dns.RR{
		&dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    ttl,
			},
			A: a.To4(),
		},
	}

	switch q.Qtype {
	case dns.TypeA:
		m.Answer = append(m.Answer, answer...)
	default:
		m.Extra = append(m.Extra, answer...)
	}

	if ns := viper.Get("dns.ns"); ns != nil {
		for _, v := range ns.([]interface{}) {
			vv := v.(map[string]interface{})
			ns := vv["ns"].(string)
			ip := vv["ip"].(string)
			ttl := uint32(vv["ttl"].(int64))
			m.Ns = append(m.Ns, &dns.NS{
				Hdr: dns.RR_Header{
					Name:   zone,
					Rrtype: dns.TypeNS,
					Class:  dns.ClassINET,
					Ttl:    ttl,
				},
				Ns: ns,
			})
			m.Extra = append(m.Extra, &dns.A{
				Hdr: dns.RR_Header{
					Name:   ns,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    ttl,
				},
				A: net.ParseIP(ip),
			})
		}
	}
}

func handleHttpPlain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/xml")
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
	}()

	badRequest := func(msg string) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(msg))
		_, _ = w.Write([]byte("\n"))
	}

	qs := r.URL.Query()
	host := qs.Get("host")
	if host == "" {
		badRequest("error. host")
		return
	}
	stime := qs.Get("time")
	if stime == "" {
		badRequest("error. time")
		return
	}
	utime, err := strconv.ParseInt(stime, 10, 32)
	if err != nil {
		badRequest("error. time")
		return
	}
	now := time.Now().Unix()
	if now-utime > 300 || utime-now > 300 {
		badRequest("error. expired")
		return
	}

	ip := qs.Get("ip")
	if ip == "" {
		ip = r.Header.Get("X-Real-Ip")
	}
	if ip == "" {
		remoteAddr := strings.SplitN(r.RemoteAddr, ":", 2)
		ip = remoteAddr[0]
	}
	a := net.ParseIP(ip)
	if a == nil {
		badRequest("error. ip")
		return
	}

	item, ok := data[host]
	if !ok {
		badRequest("error. host not found")
		return
	}
	if item.Key != "" {
		sign := qs.Get("sign")
		if sign == "" {
			badRequest("error. sign")
			return
		}
		sign_c := fmt.Sprintf("%x",
			sha1.Sum(
				[]byte(fmt.Sprintf("%s%s%s", host, stime, item.Key)),
			),
		)
		if sign != sign_c {
			badRequest("error. sign")
			return
		}
	}

	if item.IP != ip {
		item.IP = ip
		data[host] = item
		writeData()
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf("ok. %s\n", ip)))
}
