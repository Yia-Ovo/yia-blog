package service

import (
	"blog-server/api/dto"
	"blog-server/common/consts"
	"blog-server/domain/do"
	"blog-server/domain/repo"
	domainservice "blog-server/domain/service"
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// WeatherCurrentCommand 的 ClientIP 来自 Handler 的可信代理解析，不来自客户端查询参数。
type WeatherCurrentCommand struct{ City, ClientIP string }

type weatherCacheEntry struct {
	value   dto.WeatherCurrentResp
	expires time.Time
}
type weatherFlight struct {
	done  chan struct{}
	value dto.WeatherCurrentResp
}

// WeatherAppService 使用进程内有界缓存和并发共享，外部服务故障不影响首页其他 API。
// IP 只存在当前加载参数中，缓存键使用摘要；缓存与上游预算在进程重启后重置。
type WeatherAppService struct {
	domain                            *domainservice.WeatherDomainService
	provider                          repo.WeatherProvider
	mu                                sync.Mutex
	cache                             map[string]weatherCacheEntry
	flights                           map[string]*weatherFlight
	now                               func() time.Time
	minuteLoads, dayIPLoads, dayLoads []time.Time
}

func NewWeatherAppService(domain *domainservice.WeatherDomainService, provider repo.WeatherProvider) *WeatherAppService {
	return &WeatherAppService{domain: domain, provider: provider, cache: make(map[string]weatherCacheEntry), flights: make(map[string]*weatherFlight), now: time.Now}
}

func (s *WeatherAppService) Current(ctx context.Context, command WeatherCurrentCommand) (*dto.WeatherCurrentResp, error) {
	selection, err := s.domain.Select(command.City, command.ClientIP)
	if err != nil {
		return nil, err
	}
	if selection.Location == nil && selection.IP == "" {
		value := weatherUnavailable(consts.WeatherLocationUnavailable)
		return &value, nil
	}
	key := "city:" + selection.CityID
	if selection.IP != "" {
		key = fmt.Sprintf("ip:%x", sha256.Sum256([]byte(selection.IP)))
	}
	s.mu.Lock()
	now := s.now()
	if entry, ok := s.cache[key]; ok {
		if now.Before(entry.expires) {
			value := cloneWeatherResponse(entry.value)
			s.mu.Unlock()
			return &value, nil
		}
		delete(s.cache, key)
	}
	flight, exists := s.flights[key]
	if !exists {
		// 只移除已经过期的加载，避免固定窗口重置前后的两批请求突破提供方额度。
		s.minuteLoads = weatherRecentLoads(s.minuteLoads, now, time.Minute)
		s.dayIPLoads = weatherRecentLoads(s.dayIPLoads, now, 24*time.Hour)
		s.dayLoads = weatherRecentLoads(s.dayLoads, now, 24*time.Hour)
		if len(s.flights) >= consts.WeatherMaxFlights || len(s.minuteLoads) >= consts.WeatherLoadsPerMinute || len(s.dayLoads) >= consts.WeatherTotalLoadsPerDay ||
			(selection.IP != "" && len(s.dayIPLoads) >= consts.WeatherIPLoadsPerDay) {
			s.mu.Unlock()
			value := weatherUnavailable(consts.WeatherBusy)
			return &value, nil
		}
		s.minuteLoads = append(s.minuteLoads, now)
		s.dayLoads = append(s.dayLoads, now)
		if selection.IP != "" {
			s.dayIPLoads = append(s.dayIPLoads, now)
		}
		flight = &weatherFlight{done: make(chan struct{})}
		s.flights[key] = flight
		// 独立且有界的 context：一个访问者取消请求，不会中断同键其他等待者。
		go s.load(key, selection, flight)
	}
	s.mu.Unlock()
	select {
	case <-ctx.Done():
		value := weatherUnavailable(consts.WeatherBusy)
		return &value, nil
	case <-flight.done:
		value := cloneWeatherResponse(flight.value)
		return &value, nil
	}
}

func (s *WeatherAppService) load(key string, selection do.WeatherSelectionDO, flight *weatherFlight) {
	ctx, cancel := context.WithTimeout(context.Background(), consts.WeatherLoadTimeout)
	defer cancel()
	value := s.loadValue(ctx, selection)
	ttl := consts.WeatherCacheTTL
	if value.Status != consts.WeatherReady {
		ttl = consts.WeatherFailureTTL
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for entryKey, entry := range s.cache {
		if !now.Before(entry.expires) {
			delete(s.cache, entryKey)
		}
	}
	if len(s.cache) >= consts.WeatherCacheEntries {
		var oldest string
		var expiry time.Time
		for entryKey, entry := range s.cache {
			if oldest == "" || entry.expires.Before(expiry) {
				oldest = entryKey
				expiry = entry.expires
			}
		}
		delete(s.cache, oldest)
	}
	s.cache[key] = weatherCacheEntry{value: value, expires: now.Add(ttl)}
	flight.value = value
	delete(s.flights, key)
	close(flight.done)
}

func (s *WeatherAppService) loadValue(ctx context.Context, selection do.WeatherSelectionDO) dto.WeatherCurrentResp {
	var location do.WeatherLocationDO
	if selection.Location != nil {
		location = *selection.Location
	} else {
		located, err := s.provider.LocateIP(ctx, selection.IP)
		if err != nil {
			return weatherUnavailable(consts.WeatherLocationUnavailable)
		}
		var ok bool
		location, ok = s.domain.NormalizeIPLocation(located)
		if !ok {
			return weatherUnavailable(consts.WeatherLocationUnavailable)
		}
	}
	result := weatherUnavailable(consts.WeatherDataUnavailable)
	result.Location = &dto.WeatherLocationResp{City: location.City, Region: location.Region, Country: location.Country, Source: location.Source}
	weather, err := s.provider.Current(ctx, location)
	if err != nil {
		return result
	}
	result.Status = consts.WeatherReady
	result.Reason = ""
	result.Weather = &dto.WeatherDataResp{Temperature: weather.Temperature, Code: weather.Code, IsDay: weather.IsDay, Humidity: weather.Humidity,
		WindSpeed: weather.WindSpeed, Minimum: weather.Minimum, Maximum: weather.Maximum, UpdatedAt: weather.UpdatedAt}
	return result
}

func weatherUnavailable(reason string) dto.WeatherCurrentResp {
	return dto.WeatherCurrentResp{Status: consts.WeatherUnavailable, Reason: reason}
}

func cloneWeatherResponse(value dto.WeatherCurrentResp) dto.WeatherCurrentResp {
	if value.Location != nil {
		location := *value.Location
		value.Location = &location
	}
	if value.Weather != nil {
		weather := *value.Weather
		value.Weather = &weather
	}
	return value
}

// weatherRecentLoads 在持锁状态下淘汰窗口外记录，数量由各预算上限约束。
func weatherRecentLoads(loads []time.Time, now time.Time, window time.Duration) []time.Time {
	first := 0
	for first < len(loads) && now.Sub(loads[first]) >= window {
		first++
	}
	if first == 0 {
		return loads
	}
	remaining := copy(loads, loads[first:])
	clear(loads[remaining:])
	return loads[:remaining]
}
