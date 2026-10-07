package repo

import (
	"blog-server/common/consts"
	"blog-server/domain/do"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type weatherRoundTrip func(*http.Request) (*http.Response, error)

func (f weatherRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func weatherHTTPResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func weatherFixture(now time.Time) map[string]any {
	return map[string]any{
		"current":       map[string]any{"time": now.Unix(), "temperature_2m": 20, "relative_humidity_2m": 70, "weather_code": 0, "is_day": 1, "wind_speed_10m": 3},
		"current_units": map[string]any{"temperature_2m": "°C", "relative_humidity_2m": "%", "wind_speed_10m": "km/h"},
		"daily":         map[string]any{"time": []int64{now.Truncate(24 * time.Hour).Unix()}, "temperature_2m_min": []float64{15}, "temperature_2m_max": []float64{25}},
		"daily_units":   map[string]any{"temperature_2m_min": "°C", "temperature_2m_max": "°C"},
	}
}
func TestWeatherProviderFixedHostsAndSchema(t *testing.T) {
	now := time.Now().UTC()
	r := NewWeatherHTTPRepo()
	r.now = func() time.Time { return now }
	var visited []string
	r.client.Transport = weatherRoundTrip(func(request *http.Request) (*http.Response, error) {
		if request.URL.Scheme != "https" {
			t.Fatal("non-HTTPS")
		}
		visited = append(visited, request.URL.Host)
		switch request.URL.Host {
		case "ipwho.is":
			if request.URL.Path != "/8.8.8.8" || request.URL.Query().Get("fields") == "" {
				t.Fatal("unexpected IP request")
			}
			return weatherHTTPResponse(`{"success":true,"city":"City","region":"Region","country":"Country","latitude":22.8,"longitude":108.3}`), nil
		case "api.open-meteo.com":
			if request.URL.Query().Get("timezone") != "auto" || request.URL.Query().Get("temperature_unit") != "celsius" {
				t.Fatal("missing forecast constraints")
			}
			body, _ := json.Marshal(weatherFixture(now))
			return weatherHTTPResponse(string(body)), nil
		default:
			t.Fatal("arbitrary host")
			return nil, nil
		}
	})
	location, err := r.LocateIP(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	weather, err := r.Current(context.Background(), location)
	if err != nil || weather.Temperature != 20 || !weather.IsDay || len(visited) != 2 {
		t.Fatalf("unexpected result %+v %v", weather, err)
	}
}

func TestWeatherProviderRejectsIncompleteStaleAndWrongUnits(t *testing.T) {
	now := time.Now().UTC()
	for _, name := range []string{"null", "wrong_units", "stale", "bad_day", "bad_range", "bad_code", "bad_day_time", "missing_daily"} {
		t.Run(name, func(t *testing.T) {
			fixture := weatherFixture(now)
			current := fixture["current"].(map[string]any)
			switch name {
			case "null":
				current["temperature_2m"] = nil
			case "wrong_units":
				fixture["current_units"].(map[string]any)["temperature_2m"] = "°F"
			case "stale":
				current["time"] = now.Add(-3 * time.Hour).Unix()
			case "bad_day":
				current["is_day"] = 2
			case "bad_range":
				current["relative_humidity_2m"] = 101
			case "bad_code":
				current["weather_code"] = 4
			case "bad_day_time":
				fixture["daily"].(map[string]any)["time"] = []int64{-9223372036854775808}
			case "missing_daily":
				delete(fixture, "daily")
			}
			body, _ := json.Marshal(fixture)
			r := NewWeatherHTTPRepo()
			r.now = func() time.Time { return now }
			r.client.Transport = weatherRoundTrip(func(*http.Request) (*http.Response, error) { return weatherHTTPResponse(string(body)), nil })
			if _, err := r.Current(context.Background(), do.WeatherLocationDO{Latitude: 22, Longitude: 108}); err == nil {
				t.Fatal("invalid provider data accepted")
			}
		})
	}
}

func TestWeatherProviderLimitsBodiesRejectsRedirectsAndMissingCoordinates(t *testing.T) {
	for _, name := range []string{"oversized", "trailing", "missing_coordinates", "provider_failure", "rate_limit", "redirect"} {
		t.Run(name, func(t *testing.T) {
			r := NewWeatherHTTPRepo()
			calls := 0
			r.client.Transport = weatherRoundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				response := weatherHTTPResponse(`{"success":true,"city":"City","country":"Country"}`)
				switch name {
				case "oversized":
					response = weatherHTTPResponse(strings.Repeat(" ", consts.WeatherMaxResponseBytes+1))
				case "trailing":
					response = weatherHTTPResponse(`{}{}`)
				case "provider_failure":
					response = weatherHTTPResponse(`{"success":false}`)
				case "rate_limit":
					response.StatusCode = 429
				case "redirect":
					response.StatusCode = 302
					response.Header.Set("Location", "http://127.0.0.1/internal")
				}
				return response, nil
			})
			if _, err := r.LocateIP(context.Background(), "8.8.8.8"); err == nil {
				t.Fatal("invalid provider response accepted")
			}
			if calls != 1 {
				t.Fatal("redirect followed")
			}
		})
	}
}

func TestWeatherProviderHonorsCancellation(t *testing.T) {
	r := NewWeatherHTTPRepo()
	r.client.Transport = weatherRoundTrip(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.LocateIP(ctx, "8.8.8.8"); err == nil {
		t.Fatal("cancelled upstream accepted")
	}
}
