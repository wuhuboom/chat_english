package tools

import "testing"

func TestParseQQZengCityInfo(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want *CityInfo
	}{
		{name: "city", raw: "unused|中国|云南|普洱|移动", want: &CityInfo{CountryName: "中国", RegionName: "云南", CityName: "普洱"}},
		{name: "reserved", raw: "unused|保留|||", want: &CityInfo{CountryName: "Internal network"}},
		{name: "empty", raw: "", want: nil},
		{name: "short", raw: "unused|中国", want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := parseQQZengCityInfo(test.raw)
			if test.want == nil {
				if got != nil {
					t.Fatalf("parseQQZengCityInfo(%q) = %#v, want nil", test.raw, got)
				}
				return
			}
			if got == nil || *got != *test.want {
				t.Fatalf("parseQQZengCityInfo(%q) = %#v, want %#v", test.raw, got, test.want)
			}
		})
	}
}

func TestFormatCityAddressDeduplicatesParts(t *testing.T) {
	tests := []struct {
		name string
		info *CityInfo
		want string
	}{
		{name: "country duplicated", info: &CityInfo{CountryName: "China", RegionName: "China"}, want: "China"},
		{name: "municipality duplicated", info: &CityInfo{CountryName: "中国", RegionName: "天津", CityName: "天津"}, want: "中国天津"},
		{name: "full city", info: &CityInfo{CountryName: "中国", RegionName: "云南", CityName: "普洱"}, want: "中国云南普洱"},
		{name: "nil", info: nil, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := FormatCityAddress(test.info); got != test.want {
				t.Fatalf("FormatCityAddress(%#v) = %q, want %q", test.info, got, test.want)
			}
		})
	}
}

func TestParseIpNewRejectsInvalidAndIPv6(t *testing.T) {
	for _, ip := range []string{"", "not-an-ip", "::1"} {
		if got := ParseIpNew(ip); got != nil {
			t.Fatalf("ParseIpNew(%q) = %#v, want nil", ip, got)
		}
	}
}
