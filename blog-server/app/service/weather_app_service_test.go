package service

import (
	"blog-server/api/dto"
	"blog-server/common/consts"
	"blog-server/domain/do"
	domainservice "blog-server/domain/service"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type weatherProviderFake struct {
	locates  atomic.Int32
	currents atomic.Int32
	start    chan struct{}
	release  chan struct{}
	fail     bool
}

func (p *weatherProviderFake) LocateIP(context.Context, string) (do.WeatherLocationDO, error) {
	p.locates.Add(1)
	return do.WeatherLocationDO{City: "City", Country: "Country", Latitude: 1, Longitude: 2}, nil
}
func (p *weatherProviderFake) Current(ctx context.Context, _ do.WeatherLocationDO) (do.WeatherDO, error) {
	p.currents.Add(1)
	if p.start != nil {
		select {
		case p.start <- struct{}{}:
		default:
		}
	}
	if p.release != nil {
		select {
		case <-p.release:
		case <-ctx.Done():
			return do.WeatherDO{}, ctx.Err()
		}
	}
	if p.fail {
		return do.WeatherDO{}, errors.New("upstream")
	}
	return do.WeatherDO{Temperature: 20, Minimum: 15, Maximum: 25, Code: 0, UpdatedAt: time.Now()}, nil
}

func TestWeatherCachePrivateSkipAndResponseIsolation(t *testing.T) {
	p := &weatherProviderFake{}
	s := NewWeatherAppService(domainservice.NewWeatherDomainService(), p)
	result, err := s.Current(context.Background(), WeatherCurrentCommand{ClientIP: "127.0.0.1"})
	if err != nil || result.Reason != consts.WeatherLocationUnavailable || p.locates.Load() != 0 {
		t.Fatal("private IP must not leave server")
	}
	command := WeatherCurrentCommand{ClientIP: "8.8.8.8"}
	first, _ := s.Current(context.Background(), command)
	first.Location.City = "changed"
	first.Weather.Temperature = 999
	second, _ := s.Current(context.Background(), command)
	if second.Location.City == "changed" || second.Weather.Temperature == 999 || p.currents.Load() != 1 || p.locates.Load() != 1 {
		t.Fatal("cache alias or reload")
	}
	for key := range s.cache {
		if key == "ip:8.8.8.8" {
			t.Fatal("plaintext IP in cache key")
		}
	}
}

func TestWeatherConcurrentLoadSurvivesFirstWaiterCancellation(t *testing.T) {
	p := &weatherProviderFake{start: make(chan struct{}, 1), release: make(chan struct{})}
	s := NewWeatherAppService(domainservice.NewWeatherDomainService(), p)
	ctx, cancel := context.WithCancel(context.Background())
	firstDone := make(chan struct{})
	go func() { defer close(firstDone); _, _ = s.Current(ctx, WeatherCurrentCommand{City: "nanning"}) }()
	<-p.start
	cancel()
	<-firstDone
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := s.Current(context.Background(), WeatherCurrentCommand{City: "nanning"})
			if err != nil || result.Status != consts.WeatherReady {
				t.Error("shared result failed")
			}
		}()
	}
	close(p.release)
	wg.Wait()
	if p.currents.Load() != 1 {
		t.Fatalf("expected one shared load got %d", p.currents.Load())
	}
}

func TestWeatherNegativeCacheExpiresAndManualSelectionSkipsIP(t *testing.T) {
	p := &weatherProviderFake{fail: true}
	s := NewWeatherAppService(domainservice.NewWeatherDomainService(), p)
	now := time.Now()
	s.now = func() time.Time { return now }
	command := WeatherCurrentCommand{City: "tokyo"}
	for range 2 {
		result, _ := s.Current(context.Background(), command)
		if result.Reason != consts.WeatherDataUnavailable {
			t.Fatal("expected unavailable")
		}
	}
	if p.currents.Load() != 1 || p.locates.Load() != 0 {
		t.Fatal("failure not cached or manual IP requested")
	}
	now = now.Add(consts.WeatherFailureTTL + time.Second)
	p.fail = false
	result, _ := s.Current(context.Background(), command)
	if result.Status != consts.WeatherReady || p.currents.Load() != 2 {
		t.Fatal("negative cache did not expire")
	}
}

func TestWeatherCacheBoundAndLoadBudgets(t *testing.T) {
	p := &weatherProviderFake{}
	s := NewWeatherAppService(domainservice.NewWeatherDomainService(), p)
	now := time.Now()
	s.now = func() time.Time { return now }
	// 已有缓存不能无限扩张，新键加载只淘汰一条最早到期项。
	for i := range consts.WeatherCacheEntries {
		s.cache[fmt.Sprint(i)] = weatherCacheEntry{expires: now.Add(time.Hour)}
	}
	_, _ = s.Current(context.Background(), WeatherCurrentCommand{City: "tokyo"})
	if len(s.cache) != consts.WeatherCacheEntries {
		t.Fatal("cache capacity exceeded")
	}
	s.minuteLoads = weatherLoadTimes(consts.WeatherLoadsPerMinute, now)
	result, _ := s.Current(context.Background(), WeatherCurrentCommand{City: "hanoi"})
	if result.Reason != consts.WeatherBusy {
		t.Fatal("minute budget not applied")
	}
	now = now.Add(time.Minute)
	s.dayIPLoads = weatherLoadTimes(consts.WeatherIPLoadsPerDay, now)
	result, _ = s.Current(context.Background(), WeatherCurrentCommand{ClientIP: "8.8.8.8"})
	if result.Reason != consts.WeatherBusy || p.locates.Load() != 0 {
		t.Fatal("daily IP budget not applied")
	}
	result, _ = s.Current(context.Background(), WeatherCurrentCommand{City: "hanoi"})
	if result.Status != consts.WeatherReady {
		t.Fatal("IP budget should not block manual weather")
	}
}

// TestWeatherFlightCapacity 不同城市并发请求也不能突破总上游并发数。
func TestWeatherFlightCapacity(t *testing.T) {
	p := &weatherProviderFake{start: make(chan struct{}, 16), release: make(chan struct{})}
	s := NewWeatherAppService(domainservice.NewWeatherDomainService(), p)
	cities := []string{"nanning", "beijing", "shanghai", "guangzhou", "chengdu", "tokyo", "hanoi", "london"}
	var wg sync.WaitGroup
	for _, city := range cities {
		wg.Add(1)
		go func(city string) {
			defer wg.Done()
			_, _ = s.Current(context.Background(), WeatherCurrentCommand{City: city})
		}(city)
		<-p.start
	}
	value, _ := s.Current(context.Background(), WeatherCurrentCommand{City: "new-york"})
	if value.Reason != consts.WeatherBusy || p.currents.Load() != 8 {
		t.Fatal("flight limit not enforced")
	}
	close(p.release)
	wg.Wait()
}

func TestWeatherTotalDailyBudgetAlsoAppliesToManualCity(t *testing.T) {
	s := NewWeatherAppService(domainservice.NewWeatherDomainService(), &weatherProviderFake{})
	now := time.Now()
	s.now = func() time.Time { return now }
	s.dayLoads = weatherLoadTimes(consts.WeatherTotalLoadsPerDay-1, now)
	s.dayLoads = append([]time.Time{now.Add(-23 * time.Hour)}, s.dayLoads...)
	value, _ := s.Current(context.Background(), WeatherCurrentCommand{City: "tokyo"})
	if value.Reason != consts.WeatherBusy {
		t.Fatal("total daily budget not applied")
	}
	s.now = func() time.Time { return now.Add(time.Hour) }
	value, _ = s.Current(context.Background(), WeatherCurrentCommand{City: "tokyo"})
	if value.Status != consts.WeatherReady {
		t.Fatal("expired first load should release one total-budget slot")
	}
	value, _ = s.Current(context.Background(), WeatherCurrentCommand{City: "hanoi"})
	if value.Reason != consts.WeatherBusy {
		t.Fatal("recent manual loads disappeared at the daily boundary")
	}
}

// TestWeatherBudgetsDoNotResetRecentLoads 旧窗口起点过期不能清空窗口末尾的加载。
func TestWeatherBudgetsDoNotResetRecentLoads(t *testing.T) {
	for _, window := range []time.Duration{time.Minute, 24 * time.Hour} {
		t.Run(window.String(), func(t *testing.T) {
			p := &weatherProviderFake{}
			s := NewWeatherAppService(domainservice.NewWeatherDomainService(), p)
			base := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
			now := base
			s.now = func() time.Time { return now }
			request := func(index int) *dto.WeatherCurrentResp {
				t.Helper()
				value, err := s.Current(context.Background(), WeatherCurrentCommand{ClientIP: fmt.Sprintf("8.8.%d.%d", index/256, index%256)})
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
			limit := consts.WeatherLoadsPerMinute
			if window == 24*time.Hour {
				limit = consts.WeatherIPLoadsPerDay
			}
			if request(0).Status != consts.WeatherReady {
				t.Fatal("initial load failed")
			}
			for index := 1; index < limit; index++ {
				now = base.Add(window - time.Second)
				if window == 24*time.Hour {
					// 最后一个小时分散发起，避免先撞到分钟上限。
					now = base.Add(23*time.Hour + time.Duration(index/consts.WeatherLoadsPerMinute)*time.Minute)
				}
				if request(index).Status != consts.WeatherReady {
					t.Fatalf("load %d unexpectedly blocked", index)
				}
			}
			now = base.Add(window)
			if request(limit).Status != consts.WeatherReady {
				t.Fatal("expired first load should release one slot")
			}
			if request(limit+1).Reason != consts.WeatherBusy {
				t.Fatal("recent loads disappeared at the old fixed-window boundary")
			}
			if request(limit).Status != consts.WeatherReady || int(p.locates.Load()) != limit+1 {
				t.Fatal("cache hits should remain available without consuming another slot")
			}
		})
	}
}

func weatherLoadTimes(count int, at time.Time) []time.Time {
	loads := make([]time.Time, count)
	for index := range loads {
		loads[index] = at
	}
	return loads
}
