package ccnn

import (
	"context"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"net"
)

// 效果 [INFO] plugin/<name>
// [INFO] plugin/ccnn: 得到dns请求
var log = clog.NewWithPlugin("ccnn")

// Example is an ccnn plugin to show how to write a plugin.
type Ccnn struct {
	// 调用下一个插件
	Next plugin.Handler
}

var MsgIP []net.IP

// ServeDNS implements the plugin.Handler interface. This method gets called when ccnn is used
// in a Server.
// 插件核心逻辑
func (e Ccnn) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {

	if (Domain != "") && (Domain+"." == r.Question[0].Name) {

		//# HELP coredns_ccnn_intersection_ip_len ccnn intersection len of ip
		//# TYPE coredns_ccnn_intersection_ip_len gauge
		//coredns_ccnn_intersection_ip_len{domin="google.com",server="dns://:1054"} 0

		requestCount := promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: plugin.Namespace,
			Subsystem: "ccnn",
			Name:      "intersection_ip_len",
			Help:      "ccnn intersection len of ip",
			ConstLabels: map[string]string{
				"domin": Domain,
			},
		}, []string{"server"})

		state := request.Request{W: w, Req: r}

		log.Info("得到dns请求：", r.Question[0].Name)

		// 执行dns查询请求
		iprecords, _ := net.LookupIP(r.Question[0].Name)
		//for _, ip := range iprecords {
		//	log.Info(ip)
		//}
		log.Infof("实时查询结果：", iprecords)

		// 合并实时查询和配置的IP列表 取交集
		// 长度为0时异常处理
		MsgIP = intersection(Ips, iprecords)

		requestCount.WithLabelValues(metrics.WithServer(ctx)).Set(float64(len(MsgIP)))

		if len(MsgIP) > 0 {

			log.Infof("取交集的IP列表：", MsgIP, len(MsgIP))

			// 返回交集的查询结果
			// 准备相应内容

			var rr dns.RR

			switch state.Family() {
			case 1:
				rr = new(dns.A)
				rr.(*dns.A).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass()}
				rr.(*dns.A).A = MsgIP[0]
				//case 2:
				//	rr = new(dns.AAAA)
				//	rr.(*dns.AAAA).Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeAAAA, Class: state.QClass()}
				//	rr.(*dns.AAAA).AAAA = net.ParseIP(ip)
			}

			// 准备响应
			formerr := new(dns.Msg)
			formerr.SetReply(r)
			formerr.Authoritative = true
			formerr.Answer = append(formerr.Answer, rr)

			err := w.WriteMsg(formerr)
			if err != nil {
				log.Info(err)
			}

		} else {
			formerr := new(dns.Msg)
			//;; global options: +cmd
			//;; Got answer:
			//;; ->>HEADER<<- opcode: QUERY, status: NXDOMAIN, id: 46451
			//;; flags: qr rd; QUERY: 1, ANSWER: 0, AUTHORITY: 0, ADDITIONAL: 1
			//;; WARNING: recursion requested but not available
			// status 就是 RcodeNameError
			formerr.SetRcode(state.Req, dns.RcodeNameError)
			w.WriteMsg(formerr)
			log.Infof("无结果，建议修改配置")
		}

	}

	// Call next plugin (if any).
	return plugin.NextOrFailure(e.Name(), e.Next, ctx, w, r)
}

// Name implements the Handler interface.
func (e Ccnn) Name() string {
	return "ccnn"
}

func intersection(a, b []net.IP) []net.IP {

	var rs []net.IP

	for _, ipb := range b {
		for _, ipa := range a {
			if ipb.String() == ipa.String() {
				rs = append(rs, ipa)
			}
		}
	}

	return rs
}
