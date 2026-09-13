package tools

import (
	"errors"
	"github.com/ipipdotnet/ipdb-go"
	"github.com/oschwald/geoip2-golang"
	"net"
	"strings"
)

func ParseIp(myip string) *ipdb.CityInfo {
	db, err := ipdb.NewCity("./config/city.free.ipdb")
	if err != nil {
		return nil
	}
	db.Reload("./config/city.free.ipdb")
	c, err := db.FindInfo(myip, "CN")
	if err != nil {
		return nil
	}
	return c
}
func ParseIpNew(myip string) *CityInfo {
	ip := net.ParseIP(strings.TrimSpace(myip))
	if ip == nil || ip.To4() == nil {
		return nil
	}
	p, err := NewIpdb("./config/qqzeng-ip-utf8.dat")
	if err != nil {
		return nil
	}
	return parseQQZengCityInfo(p.Get(ip.String()))
}

func parseQQZengCityInfo(raw string) *CityInfo {
	infos := strings.Split(raw, "|")
	if len(infos) < 4 {
		return nil
	}
	country := strings.TrimSpace(infos[1])
	if country == "" {
		return nil
	}
	if country == "保留" {
		return &CityInfo{CountryName: "Internal network"}
	}
	return &CityInfo{
		CountryName: country,
		RegionName:  strings.TrimSpace(infos[2]),
		CityName:    strings.TrimSpace(infos[3]),
	}
}

func ParseIPLocation(myip string) *CityInfo {
	if cityInfo := ParseIpNew(myip); cityInfo != nil {
		return cityInfo
	}
	return ParseIpNew2(myip)
}

func FormatCityAddress(cityInfo *CityInfo) string {
	if cityInfo == nil {
		return ""
	}
	seen := make(map[string]struct{}, 3)
	parts := make([]string, 0, 3)
	for _, value := range []string{cityInfo.CountryName, cityInfo.RegionName, cityInfo.CityName} {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		parts = append(parts, value)
	}
	return strings.Join(parts, "")
}
func GetServerIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ip := getIpFromAddr(addr)
			if ip == nil {
				continue
			}
			return ip, nil
		}
	}
	return nil, errors.New("connected to the network?")
}
func getIpFromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}
	if ip == nil || ip.IsLoopback() {
		return nil
	}
	ip = ip.To4()
	if ip == nil {
		return nil // not an ipv4 address
	}

	return ip
}

func ParseIpNew2(myip string) *CityInfo {
	db, err := geoip2.Open("./config/GeoLite2-Country.mmdb")
	if err != nil {
		return nil
	}
	defer db.Close()
	ip := net.ParseIP(strings.TrimSpace(myip))
	if ip == nil {
		return nil
	}
	record, err := db.City(ip)
	if err != nil {
		return nil
	}
	countryName := record.Country.Names["zh-CN"]
	if countryName == "" {
		countryName = record.Country.Names["en"]
	}
	if countryName == "" {
		countryName = record.RegisteredCountry.Names["zh-CN"]
	}
	if countryName == "" {
		countryName = record.RegisteredCountry.Names["en"]
	}
	if countryName == "" {
		return nil
	}
	return &CityInfo{CountryName: countryName}
}

// 获取出站IP地址
func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}
