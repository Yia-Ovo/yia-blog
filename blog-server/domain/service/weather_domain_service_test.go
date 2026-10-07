package service

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"math"
	"testing"
)

func TestWeatherLocationSelection(t *testing.T) {
	s := NewWeatherDomainService()
	for _, ip := range []string{"", "invalid", "127.0.0.1", "::1", "10.2.3.4", "172.16.1.2", "192.168.0.1", "169.254.169.254", "100.64.1.2", "192.0.2.1", "198.18.1.1", "203.0.113.1", "2001:db8::1", "fc00::1", "fe80::1%eth0", "224.0.0.1", "::ffff:127.0.0.1"} {
		selected, err := s.Select("", ip)
		if err != nil || selected.IP != "" || selected.Location != nil {
			t.Errorf("nonpublic %q accepted: %+v", ip, selected)
		}
	}
	for _, ip := range []string{"8.8.8.8", "2606:4700:4700::1111"} {
		selected, err := s.Select("", ip)
		if err != nil || selected.IP != ip {
			t.Errorf("public %q rejected", ip)
		}
	}
	selected, err := s.Select("nanning", "invalid")
	if err != nil || selected.Location == nil || selected.Location.Source != consts.WeatherSourceManual {
		t.Fatal("manual city should override IP")
	}
	selected.Location.City = "changed"
	again, _ := s.Select("nanning", "")
	if again.Location.City == "changed" {
		t.Fatal("mutated preset through pointer")
	}
	if _, err := s.Select("https://attacker.test", "8.8.8.8"); err == nil {
		t.Fatal("arbitrary city accepted")
	}
}

func TestWeatherIPLocationValidation(t *testing.T) {
	s := NewWeatherDomainService()
	for _, location := range []do.WeatherLocationDO{
		{City: "", Country: "China"}, {City: "city", Country: "China", Latitude: 91},
		{City: "city", Country: "China", Longitude: 181}, {City: "city", Country: "China", Latitude: math.NaN()},
	} {
		if _, ok := s.NormalizeIPLocation(location); ok {
			t.Fatalf("invalid location accepted: %+v", location)
		}
	}
	location, ok := s.NormalizeIPLocation(do.WeatherLocationDO{City: " City ", Country: "Country", Latitude: 0, Longitude: 0})
	if !ok || location.City != "City" || location.Source != consts.WeatherSourceIP {
		t.Fatal("valid zero coordinates or IP source mishandled")
	}
}
