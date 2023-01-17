package ccnn

import (
	"fmt"
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"net"
)

var Ips []net.IP
var Domain string

// init方法 注册插件
func init() {

	plugin.Register("ccnn", setup)
}

func setup(c *caddy.Controller) error {

	// 插件参数处理
	c.Next() // Ignore "example" and give us the next token.
	// todo 获取配置参数
	log.Info("插件名称", c.Val())
	c.NextBlock()
	log.Info("处理域名", c.Val())
	Domain = c.Val()
	c.Next() // val is {
	c.Next()
	for {
		if c.Val() != "}" {

			ip, _, err := net.ParseCIDR(c.Val() + "/32")
			if err != nil {
				fmt.Println(err)
			}
			Ips = append(Ips, ip)
			c.Next()
		} else {
			break
		}
	}

	log.Info("获取配置文件中的IP地址列表为：", Ips)

	//  func (c *Config) AddPlugin(m plugin.Plugin) plugin最终是个接口类型

	//Handler interface {
	//ServeDNS(context.Context, dns.ResponseWriter, *dns.Msg) (int, error)
	//Name() string
	//}

	// 自定义的插件需要实现这个接口 也就是说要实现 ServeDNS 和 Name 两个方法
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Ccnn{Next: next}
	})

	// All OK, return a nil error.
	return nil
}
