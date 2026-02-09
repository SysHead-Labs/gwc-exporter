package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type gwcCollector struct {
	targetURL string
	timeout   time.Duration

	// Descriptors
	up                                   *prometheus.Desc
	started_seconds                      *prometheus.Desc
	uptime_seconds                       *prometheus.Desc
	requests_total                       *prometheus.Desc
	requests_rate_per_second             *prometheus.Desc
	untiled_wms_requests_total           *prometheus.Desc
	untiled_wms_requests_rate_per_second *prometheus.Desc
	bytes_total                          *prometheus.Desc
	bandwidth_mbps                       *prometheus.Desc
	cache_hit_ratio_percent              *prometheus.Desc
	blank_kml_html_ratio_percent         *prometheus.Desc
	peak_request_rate_per_second         *prometheus.Desc
	peak_request_rate_timestamp_seconds  *prometheus.Desc
	peak_bandwidth_mbps                  *prometheus.Desc
	peak_bandwidth_timestamp_seconds     *prometheus.Desc
	stats_delay_seconds                  *prometheus.Desc
	interval_requests                    *prometheus.Desc // label: window
	interval_rate_per_second             *prometheus.Desc // label: window
	interval_bytes                       *prometheus.Desc // label: window
	interval_bandwidth_mbps              *prometheus.Desc // label: window
	storage_info                         *prometheus.Desc // labels: config_file, local_storage

	// In-memory cache (optional)
	memcache_present             *prometheus.Desc // 1 if section present, else 0
	memcache_requests_total      *prometheus.Desc
	memcache_hit_count_total     *prometheus.Desc
	memcache_miss_count_total    *prometheus.Desc
	memcache_hit_ratio_percent   *prometheus.Desc
	memcache_miss_ratio_percent  *prometheus.Desc
	memcache_evicted_tiles_total *prometheus.Desc
	memcache_occupation_percent  *prometheus.Desc
	memcache_actual_size_bytes   *prometheus.Desc
	memcache_total_size_bytes    *prometheus.Desc

	version_build_info *prometheus.Desc // labels: version, build
}

func newGwcCollector(targetURL string, timeout time.Duration) *gwcCollector {
	const ns = "gwc"
	return &gwcCollector{
		targetURL: targetURL,
		timeout:   timeout,

		up:                                   prometheus.NewDesc(ns+"_up", "Was the last scrape of GWC status page successful.", nil, nil),
		started_seconds:                      prometheus.NewDesc(ns+"_started_seconds", "Unix timestamp when GWC reports it started.", nil, nil),
		uptime_seconds:                       prometheus.NewDesc(ns+"_uptime_seconds", "Reported uptime in seconds.", nil, nil),
		requests_total:                       prometheus.NewDesc(ns+"_requests_total", "Total number of requests.", nil, nil),
		requests_rate_per_second:             prometheus.NewDesc(ns+"_requests_rate_per_second", "Reported requests per second.", nil, nil),
		untiled_wms_requests_total:           prometheus.NewDesc(ns+"_untiled_wms_requests_total", "Total number of untiled WMS requests.", nil, nil),
		untiled_wms_requests_rate_per_second: prometheus.NewDesc(ns+"_untiled_wms_requests_rate_per_second", "Reported untiled WMS requests per second.", nil, nil),
		bytes_total:                          prometheus.NewDesc(ns+"_bytes_total", "Total number of bytes served.", nil, nil),
		bandwidth_mbps:                       prometheus.NewDesc(ns+"_bandwidth_mbps", "Reported bandwidth in Mbps.", nil, nil),
		cache_hit_ratio_percent:              prometheus.NewDesc(ns+"_cache_hit_ratio_percent", "Cache hit ratio (percent of requests).", nil, nil),
		blank_kml_html_ratio_percent:         prometheus.NewDesc(ns+"_blank_kml_html_ratio_percent", "Blank/KML/HTML percent of requests.", nil, nil),
		peak_request_rate_per_second:         prometheus.NewDesc(ns+"_peak_request_rate_per_second", "Peak request rate (/s).", nil, nil),
		peak_request_rate_timestamp_seconds:  prometheus.NewDesc(ns+"_peak_request_rate_timestamp_seconds", "Unix timestamp when peak request rate was observed.", nil, nil),
		peak_bandwidth_mbps:                  prometheus.NewDesc(ns+"_peak_bandwidth_mbps", "Peak bandwidth in Mbps.", nil, nil),
		peak_bandwidth_timestamp_seconds:     prometheus.NewDesc(ns+"_peak_bandwidth_timestamp_seconds", "Unix timestamp when peak bandwidth was observed.", nil, nil),
		stats_delay_seconds:                  prometheus.NewDesc(ns+"_stats_delay_seconds", "Reported delay in runtime statistics.", nil, nil),

		interval_requests:        prometheus.NewDesc(ns+"_interval_requests", "Requests in the reported time window.", []string{"window"}, nil),
		interval_rate_per_second: prometheus.NewDesc(ns+"_interval_rate_per_second", "Requests per second in the reported time window.", []string{"window"}, nil),
		interval_bytes:           prometheus.NewDesc(ns+"_interval_bytes", "Bytes in the reported time window.", []string{"window"}, nil),
		interval_bandwidth_mbps:  prometheus.NewDesc(ns+"_interval_bandwidth_mbps", "Bandwidth Mbps in the reported time window.", []string{"window"}, nil),

		storage_info: prometheus.NewDesc(ns+"_storage_info", "Storage paths as labels.", []string{"config_file", "local_storage"}, nil),

		memcache_present:             prometheus.NewDesc(ns+"_memcache_present", "1 if 'In Memory Cache Statistics' section is present, else 0.", nil, nil),
		memcache_requests_total:      prometheus.NewDesc(ns+"_memcache_requests_total", "In-memory cache total number of requests.", nil, nil),
		memcache_hit_count_total:     prometheus.NewDesc(ns+"_memcache_hit_count_total", "In-memory cache hit count.", nil, nil),
		memcache_miss_count_total:    prometheus.NewDesc(ns+"_memcache_miss_count_total", "In-memory cache miss count.", nil, nil),
		memcache_hit_ratio_percent:   prometheus.NewDesc(ns+"_memcache_hit_ratio_percent", "In-memory cache hit ratio percent.", nil, nil),
		memcache_miss_ratio_percent:  prometheus.NewDesc(ns+"_memcache_miss_ratio_percent", "In-memory cache miss ratio percent.", nil, nil),
		memcache_evicted_tiles_total: prometheus.NewDesc(ns+"_memcache_evicted_tiles_total", "Total number of evicted tiles.", nil, nil),
		memcache_occupation_percent:  prometheus.NewDesc(ns+"_memcache_occupation_percent", "Cache memory occupation percent.", nil, nil),
		memcache_actual_size_bytes:   prometheus.NewDesc(ns+"_memcache_actual_size_bytes", "Cache actual size in bytes.", nil, nil),
		memcache_total_size_bytes:    prometheus.NewDesc(ns+"_memcache_total_size_bytes", "Cache total size in bytes.", nil, nil),

		version_build_info: prometheus.NewDesc(ns+"_build_info", "Version/build info as labels; value 1.", []string{"version", "build"}, nil),
	}
}

func (c *gwcCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up
	ch <- c.started_seconds
	ch <- c.uptime_seconds
	ch <- c.requests_total
	ch <- c.requests_rate_per_second
	ch <- c.untiled_wms_requests_total
	ch <- c.untiled_wms_requests_rate_per_second
	ch <- c.bytes_total
	ch <- c.bandwidth_mbps
	ch <- c.cache_hit_ratio_percent
	ch <- c.blank_kml_html_ratio_percent
	ch <- c.peak_request_rate_per_second
	ch <- c.peak_request_rate_timestamp_seconds
	ch <- c.peak_bandwidth_mbps
	ch <- c.peak_bandwidth_timestamp_seconds
	ch <- c.stats_delay_seconds
	ch <- c.interval_requests
	ch <- c.interval_rate_per_second
	ch <- c.interval_bytes
	ch <- c.interval_bandwidth_mbps
	ch <- c.storage_info

	// memcache
	ch <- c.memcache_present
	ch <- c.memcache_requests_total
	ch <- c.memcache_hit_count_total
	ch <- c.memcache_miss_count_total
	ch <- c.memcache_hit_ratio_percent
	ch <- c.memcache_miss_ratio_percent
	ch <- c.memcache_evicted_tiles_total
	ch <- c.memcache_occupation_percent
	ch <- c.memcache_actual_size_bytes
	ch <- c.memcache_total_size_bytes

	ch <- c.version_build_info
}

func (c *gwcCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.targetURL, nil)
	if err != nil {
		log.Printf("gwc scrape: cannot create request target=%q err=%v", c.targetURL, err)
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("gwc scrape: request failed target=%q err=%v", c.targetURL, err)
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("gwc scrape: non-200 response target=%q status=%d", c.targetURL, resp.StatusCode)
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("gwc scrape: cannot read response body target=%q err=%v", c.targetURL, err)
		ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 0)
		return
	}
	html := string(body)

	// Normalize
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = regexp.MustCompile(`\s+`).ReplaceAllString(html, " ")

	// base liveness
	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, 1)

	// Version/build
	if ver, build := parseVersionBuild(html); ver != "" || build != "" {
		ch <- prometheus.MustNewConstMetric(c.version_build_info, prometheus.GaugeValue, 1, ver, build)
	}

	// Started + uptime
	if ts := parseStartedUnix(html); ts > 0 {
		ch <- prometheus.MustNewConstMetric(c.started_seconds, prometheus.GaugeValue, float64(ts))
	}
	if up := parseUptimeSeconds(html); up > 0 {
		ch <- prometheus.MustNewConstMetric(c.uptime_seconds, prometheus.GaugeValue, float64(up))
	}

	// Totals + rates
	if v := parseFirstNumber(html, `Total number of requests:\s*</th>\s*<td[^>]*>\s*([0-9,]+)`); v >= 0 {
		ch <- prometheus.MustNewConstMetric(c.requests_total, prometheus.CounterValue, float64(v))
	}
	//if v := parseFirstFloat(html, `Total number of requests:.*\(([0-9.]+)\/s`); v >= 0 {
	//if v := parseFirstFloat(html, `Total number of requests:\s*[0-9,]+\s*\(([0-9.]+)\s*/s\s*\)`); v >= 0 {
	//if v := parseFirstFloat(html, `Total number of requests:\s*[0-9,]+\s*\(\s*([0-9.]+)\s*/s\s*\)`); v >= 0 {
	if v := parseFirstFloat(html, `Total number of requests:\s*</th>\s*<td[^>]*>\s*[0-9,]+\s*\(\s*([0-9.]+)\s*/s\s*\)\s*</td>`); v >= 0 {
		ch <- prometheus.MustNewConstMetric(c.requests_rate_per_second, prometheus.GaugeValue, v)
	}
	//if v := parseFirstNumber(html, `Total number of untiled WMS requests:\s*([0-9]+)`); v >= 0 {
	if v := parseFirstNumber(html, `Total number of untiled WMS requests:\s*</th>\s*<td[^>]*>\s*([0-9,]+)`); v >= 0 {
		ch <- prometheus.MustNewConstMetric(c.untiled_wms_requests_total, prometheus.CounterValue, float64(v))
	}
	if v := parseFirstFloat(html, `Total number of untiled WMS requests:\s*</th>\s*<td[^>]*>\s*[0-9,]+\s*\(\s*([0-9.]+)\s*/s\s*\)\s*</td>`); v >= 0 {
		ch <- prometheus.MustNewConstMetric(c.untiled_wms_requests_rate_per_second, prometheus.GaugeValue, v)
	}
	if v := parseFirstNumber(html, `Total number of bytes:\s*</th>\s*<td[^>]*>\s*([0-9,]+)`); v >= 0 {
		ch <- prometheus.MustNewConstMetric(c.bytes_total, prometheus.CounterValue, float64(v))
	}
	if v := parseFirstFloat(html, `Total number of bytes:\s*</th>\s*<td[^>]*>\s*[0-9,]+\s*\(\s*([0-9.]+)\s*mbps`); v >= 0 {
		ch <- prometheus.MustNewConstMetric(c.bandwidth_mbps, prometheus.GaugeValue, v)
	}
	//if v := parseFirstFloat(html, `Cache hit ratio:\s*([0-9.]+)% of requests`); v >= 0 {
	if v := parseFirstFloat(html, `Cache hit ratio:\s*</th>\s*<td[^>]*>\s*([0-9.]+)% of requests`); v >= 0 {
		ch <- prometheus.MustNewConstMetric(c.cache_hit_ratio_percent, prometheus.GaugeValue, v)
	}
	if v := parseFirstFloat(html, `Blank/KML/HTML:\s*</th>\s*<td[^>]*>\s*([0-9.]+)% of requests`); v >= 0 {
		ch <- prometheus.MustNewConstMetric(c.blank_kml_html_ratio_percent, prometheus.GaugeValue, v)
	}
	if v := parseFirstFloat(html, `Peak request rate:\s*</th>\s*<td[^>]*>\s*([0-9.]+)\s*/s`); v >= 0 {
		ch <- prometheus.MustNewConstMetric(c.peak_request_rate_per_second, prometheus.GaugeValue, v)
	}
	if ts := parseRFC1123InParens(html, `Peak request rate:\s*</th>\s*<td[^>]*>\s*[0-9.]+\s*/s\s*\(([^)]+)\)`); ts > 0 {
		ch <- prometheus.MustNewConstMetric(c.peak_request_rate_timestamp_seconds, prometheus.GaugeValue, float64(ts))
	}
	if v := parseFirstFloat(html, `Peak bandwidth:\s*</th>\s*<td[^>]*>\s*([0-9.]+)\s*mbps`); v >= 0 {
		ch <- prometheus.MustNewConstMetric(c.peak_bandwidth_mbps, prometheus.GaugeValue, v)
	}
	if ts := parseRFC1123InParens(html, `Peak bandwidth:\s*</th>\s*<td[^>]*>\s*[0-9.]+\s*mbps\s*\(([^)]+)\)`); ts > 0 {
		ch <- prometheus.MustNewConstMetric(c.peak_bandwidth_timestamp_seconds, prometheus.GaugeValue, float64(ts))
	}
	if v := parseFirstFloat(html, `All figures are ([0-9.]+)\s*second\(s\) delayed`); v >= 0 {
		ch <- prometheus.MustNewConstMetric(c.stats_delay_seconds, prometheus.GaugeValue, v)
	}

	// Interval rows
	intervals := []string{"3 seconds", "15 seconds", "60 seconds"}
	for _, win := range intervals {
		re := regexp.MustCompile(fmt.Sprintf(`<tr>\s*<td>\s*%s\s*</td>\s*<td[^>]*>\s*([0-9,]+)\s*</td>\s*<td[^>]*>\s*([0-9.]+)\s*/s\s*</td>\s*<td[^>]*>\s*([0-9,]+)\s*</td>\s*<td[^>]*>\s*([0-9.]+)\s*mbps`, regexp.QuoteMeta(win)))
		if m := re.FindStringSubmatch(html); len(m) == 5 {
			reqs := toInt(m[1])
			rate := toFloat(m[2])
			bytes := toInt(m[3])
			mbps := toFloat(m[4])
			if reqs >= 0 {
				ch <- prometheus.MustNewConstMetric(c.interval_requests, prometheus.GaugeValue, float64(reqs), win)
			}
			if rate >= 0 {
				ch <- prometheus.MustNewConstMetric(c.interval_rate_per_second, prometheus.GaugeValue, rate, win)
			}
			if bytes >= 0 {
				ch <- prometheus.MustNewConstMetric(c.interval_bytes, prometheus.GaugeValue, float64(bytes), win)
			}
			if mbps >= 0 {
				ch <- prometheus.MustNewConstMetric(c.interval_bandwidth_mbps, prometheus.GaugeValue, mbps, win)
			}
		}
	}

	// Storage info
	configFile := parseFirstString(html, `Config file:\s*</th>\s*<td[^>]*>\s*<tt>([^<]+)`)
	localStorage := parseFirstString(html, `Local Storage:\s*</th>\s*<td[^>]*>\s*<tt>([^<]+)`)
	if configFile != "" || localStorage != "" {
		ch <- prometheus.MustNewConstMetric(c.storage_info, prometheus.GaugeValue, 1, configFile, localStorage)
	}

	// In-memory cache — only emit when section exists
	if strings.Contains(html, "In Memory Cache Statistics") {
		ch <- prometheus.MustNewConstMetric(c.memcache_present, prometheus.GaugeValue, 1)

		// Values are in the next <td> after the label
		if v := parseFirstNumber(html, `Total number of requests:</td><td[^>]*>\s*([0-9,]+)`); v >= 0 {
			ch <- prometheus.MustNewConstMetric(c.memcache_requests_total, prometheus.CounterValue, float64(v))
		}
		if v := parseFirstNumber(html, `Internal Cache hit count:</td><td[^>]*>\s*([0-9,]+)`); v >= 0 {
			ch <- prometheus.MustNewConstMetric(c.memcache_hit_count_total, prometheus.CounterValue, float64(v))
		}
		if v := parseFirstNumber(html, `Internal Cache miss count:</td><td[^>]*>\s*([0-9,]+)`); v >= 0 {
			ch <- prometheus.MustNewConstMetric(c.memcache_miss_count_total, prometheus.CounterValue, float64(v))
		}
		if v := parseFirstFloat(html, `Internal Cache hit ratio:</td><td[^>]*>\s*([0-9.]+)\s*%`); v >= 0 {
			ch <- prometheus.MustNewConstMetric(c.memcache_hit_ratio_percent, prometheus.GaugeValue, v)
		}
		if v := parseFirstFloat(html, `Internal Cache miss ratio:</td><td[^>]*>\s*([0-9.]+)\s*%`); v >= 0 {
			ch <- prometheus.MustNewConstMetric(c.memcache_miss_ratio_percent, prometheus.GaugeValue, v)
		}
		if v := parseFirstNumber(html, `Total number of evicted tiles:</td><td[^>]*>\s*([0-9,]+)`); v >= 0 {
			ch <- prometheus.MustNewConstMetric(c.memcache_evicted_tiles_total, prometheus.CounterValue, float64(v))
		}
		if v := parseFirstFloat(html, `Cache Memory occupation:</td><td[^>]*>\s*([0-9.]+)\s*%`); v >= 0 {
			ch <- prometheus.MustNewConstMetric(c.memcache_occupation_percent, prometheus.GaugeValue, v)
		}
		if act, tot := parseSizesMB(html); act >= 0 && tot >= 0 {
			// MB as decimal 1e6
			ch <- prometheus.MustNewConstMetric(c.memcache_actual_size_bytes, prometheus.GaugeValue, act*1e6)
			ch <- prometheus.MustNewConstMetric(c.memcache_total_size_bytes, prometheus.GaugeValue, tot*1e6)
		}
	} else {
		ch <- prometheus.MustNewConstMetric(c.memcache_present, prometheus.GaugeValue, 0)
		// Keep a stable metric set when memcache is disabled.
		ch <- prometheus.MustNewConstMetric(c.memcache_requests_total, prometheus.CounterValue, 0)
		ch <- prometheus.MustNewConstMetric(c.memcache_hit_count_total, prometheus.CounterValue, 0)
		ch <- prometheus.MustNewConstMetric(c.memcache_miss_count_total, prometheus.CounterValue, 0)
		ch <- prometheus.MustNewConstMetric(c.memcache_hit_ratio_percent, prometheus.GaugeValue, 0)
		ch <- prometheus.MustNewConstMetric(c.memcache_miss_ratio_percent, prometheus.GaugeValue, 0)
		ch <- prometheus.MustNewConstMetric(c.memcache_evicted_tiles_total, prometheus.CounterValue, 0)
		ch <- prometheus.MustNewConstMetric(c.memcache_occupation_percent, prometheus.GaugeValue, 0)
		ch <- prometheus.MustNewConstMetric(c.memcache_actual_size_bytes, prometheus.GaugeValue, 0)
		ch <- prometheus.MustNewConstMetric(c.memcache_total_size_bytes, prometheus.GaugeValue, 0)
	}
}

func parseVersionBuild(html string) (string, string) {
	re := regexp.MustCompile(`Welcome to GeoWebCache version ([^,]+), build ([^<]+)`)
	m := re.FindStringSubmatch(html)
	if len(m) == 3 {
		return strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
	}
	return "", ""
}

/*
func parseStartedUnix(html string) int64 {
	re := regexp.MustCompile(`Started:\s*([A-Za-z]{3}, [0-9]{2} [A-Za-z]{3} [0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2} GMT)`)
	m := re.FindStringSubmatch(html)
	if len(m) != 2 {
		return 0
	}
	t, err := time.Parse(time.RFC1123, m[1])
	if err != nil {
		return 0
	}
	return t.Unix()
}*/

func parseStartedUnix(html string) int64 {
	// Case 1: value in the next <td> cell after the "Started:" header
	reCell := regexp.MustCompile(`Started:\s*</th>\s*<td[^>]*>\s*([A-Za-z]{3}, [0-9]{1,2} [A-Za-z]{3} [0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2} GMT)`)
	if m := reCell.FindStringSubmatch(html); len(m) == 2 {
		if t, err := time.Parse(time.RFC1123, m[1]); err == nil {
			return t.Unix()
		}
	}

	// Case 2: plain text after "Started:"
	rePlain := regexp.MustCompile(`Started:\s*([A-Za-z]{3}, [0-9]{1,2} [A-Za-z]{3} [0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2} GMT)`)
	if m := rePlain.FindStringSubmatch(html); len(m) == 2 {
		if t, err := time.Parse(time.RFC1123, m[1]); err == nil {
			return t.Unix()
		}
	}

	return 0
}

func parseRFC1123InParens(html, pattern string) int64 {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(html)
	if len(m) < 2 {
		return 0
	}
	t, err := time.Parse(time.RFC1123, strings.TrimSpace(m[1]))
	if err != nil {
		return 0
	}
	return t.Unix()
}

func parseUptimeSeconds(html string) int64 {
	re := regexp.MustCompile(`Started:.*\(([0-9.]+)\s*(seconds|second|minutes|minute|hours|hour|days|day)\)`)
	m := re.FindStringSubmatch(html)
	if len(m) != 3 {
		return 0
	}
	val := toFloat(m[1])
	unit := strings.ToLower(m[2])
	switch {
	case strings.HasPrefix(unit, "second"):
		return int64(val)
	case strings.HasPrefix(unit, "minute"):
		return int64(val * 60)
	case strings.HasPrefix(unit, "hour"):
		return int64(val * 3600)
	case strings.HasPrefix(unit, "day"):
		return int64(val * 86400)
	default:
		return 0
	}
}

func parseSizesMB(html string) (float64, float64) {
	// Same cell
	if m := regexp.MustCompile(`Cache Actual Size/ Total Size :\s*([0-9.]+)\s*/\s*([0-9.]+)\s* Mb`).FindStringSubmatch(html); len(m) == 3 {
		return toFloat(m[1]), toFloat(m[2])
	}
	// Next <td> cell
	if m := regexp.MustCompile(`Cache Actual Size/ Total Size :\s*</td>\s*<td[^>]*>\s*([0-9.]+)\s*/\s*([0-9.]+)\s*Mb`).FindStringSubmatch(html); len(m) == 3 {
		return toFloat(m[1]), toFloat(m[2])
	}
	return -1, -1
}

func parseFirstString(html, pattern string) string {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(html)
	if len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func parseFirstNumber(html, pattern string) int64 {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(html)
	if len(m) < 2 {
		return -1
	}
	return toInt(m[1])
}

func parseFirstFloat(html, pattern string) float64 {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(html)
	if len(m) < 2 {
		return -1
	}
	return toFloat(m[1])
}

func toInt(s string) int64 {
	i, err := strconv.ParseInt(strings.ReplaceAll(s, ",", ""), 10, 64)
	if err != nil {
		return -1
	}
	return i
}

func toFloat(s string) float64 {
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil {
		return -1
	}
	return f
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("invalid %s=%q, using default %s", key, raw, fallback)
		return fallback
	}
	return v
}

func main() {
	var (
		url = flag.String(
			"target.url",
			envOrDefault("GWC_TARGET_URL", "http://127.0.0.1:8080/geowebcache"),
			"Full URL of the GeoWebCache HTML status page. Can also be set by GWC_TARGET_URL.",
		)
		addr = flag.String(
			"web.listen-address",
			envOrDefault("GWC_WEB_LISTEN_ADDRESS", ":9109"),
			"Address to listen on for /metrics. Can also be set by GWC_WEB_LISTEN_ADDRESS.",
		)
		path = flag.String(
			"web.telemetry-path",
			envOrDefault("GWC_WEB_TELEMETRY_PATH", "/metrics"),
			"Path under which to expose metrics. Can also be set by GWC_WEB_TELEMETRY_PATH.",
		)
		timeout = flag.Duration(
			"scrape.timeout",
			envDurationOrDefault("GWC_SCRAPE_TIMEOUT", 5*time.Second),
			"HTTP timeout when scraping the target URL. Can also be set by GWC_SCRAPE_TIMEOUT (e.g. 5s).",
		)
	)
	flag.Parse()

	collector := newGwcCollector(*url, *timeout)
	reg := prometheus.NewRegistry()
	if err := reg.Register(collector); err != nil {
		log.Fatalf("register collector: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle(*path, promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))

	// basic liveness
	mux.HandleFunc("/-/healthy", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("GWC exporter listening on %s, scraping %s", *addr, *url)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("http server: %v", err)
	}
}
