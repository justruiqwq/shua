package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func getEnvString(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
func getEnvBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
func getEnvBytes(key string, def int64) int64 {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := parseBytes(v); err == nil {
			return b
		}
	}
	return def
}
func parseBytes(s string) (int64, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "k"):
		mult = 1024
		s = s[:len(s)-1]
	case strings.HasSuffix(s, "m"):
		mult = 1024 * 1024
		s = s[:len(s)-1]
	case strings.HasSuffix(s, "g"):
		mult = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case strings.HasSuffix(s, "t"):
		mult = 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case strings.HasSuffix(s, "p"):
		mult = 1024 * 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}
	v, err := strconv.ParseFloat(s, 64)
	return int64(v * float64(mult)), err
}

func worker(ctx context.Context, id int, wg *sync.WaitGroup, url string,
	interval time.Duration, ua string, rateLimit int64,
	transport http.RoundTripper) {

	defer wg.Done()
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	noInterval := interval == 0
	fmt.Printf("Worker %d 启动 [间隔: %v 限速: %v]\n", id,
		map[bool]string{true: "无", false: interval.String()}[noInterval],
		func() string {
			if rateLimit <= 0 {
				return "不限"
			}
			return fmt.Sprintf("%d B/s", rateLimit)
		}())

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("Worker %d 停止\n", id)
			return
		default:
			start := time.Now()
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("User-Agent", ua)

			resp, err := client.Do(req)
			stop := handleResponse(id, resp, err, start)
			if stop {
				return
			}

			if !noInterval {
				select {
				case <-ctx.Done():
					return
				case <-time.After(interval):
				}
			}
		}
	}
}

func handleResponse(id int, resp *http.Response, err error, start time.Time) (stop bool) {
	if err != nil {
		fmt.Printf("[Worker %d][%s] 请求失败: %v\n",
			id, time.Now().Format("2006-01-02 15:04:05"), err)
		return false
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	code := resp.StatusCode
	latency := time.Since(start).Round(time.Millisecond)
	if code != http.StatusOK {
		fmt.Printf("[Worker %d][%s] 状态码: %d 延迟: %v\n",
			id, time.Now().Format("2006-01-02 15:04:05"), code, latency)
	}
	return false
}

func main() {
	var (
		url = flag.String("url",
			getEnvString("url", "https://js.a.kspkg.com/kos/nlav10814/kwai-android-generic-gifmakerrelease-13.7.30.43728_x64_5d82bf.apk"),
			"目标URL")

		interval = flag.Duration("i",
			func() time.Duration {
				if v, ok := os.LookupEnv("i"); ok {
					if d, err := time.ParseDuration(v); err == nil {
						return d
					}
				}
				return 0
			}(),
			"请求间隔")

		workers = flag.Int("w",
			func() int {
				if v, ok := os.LookupEnv("w"); ok {
					if i, err := strconv.Atoi(v); err == nil {
						return i
					}
				}
				return 64
			}(),
			"Worker 数量")

		userAgent = flag.String("ua",
			getEnvString("ua", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"),
			"User-Agent")

		address = flag.String("address", getEnvString("address", ""), "源IP地址")
		ipv4Only = flag.Bool("4", getEnvBool("4", false), "仅 IPv4")
		ipv6Only = flag.Bool("6", getEnvBool("6", false), "仅 IPv6")
		rateStr  = flag.String("rate", getEnvString("rate", ""), "限速，如 1.5m")
	)
	flag.Parse()

	if *workers <= 0 {
		fmt.Println("[错误] worker 数量必须 > 0")
		return
	}
	if *interval < 0 {
		fmt.Println("[错误] 间隔时间不能为负数")
		return
	}
	if *ipv4Only && *ipv6Only {
		fmt.Println("[错误] 不能同时指定 -4 和 -6")
		return
	}

	var localAddr *net.TCPAddr
	if *address != "" {
		ip := net.ParseIP(*address)
		if ip == nil {
			fmt.Printf("[错误] address 无效: %s\n", *address)
			return
		}
		if *ipv4Only && ip.To4() == nil {
			fmt.Printf("[错误] address 需要 IPv4: %s\n", *address)
			return
		}
		if *ipv6Only && ip.To4() != nil {
			fmt.Printf("[错误] address 需要 IPv6: %s\n", *address)
			return
		}
		localAddr = &net.TCPAddr{IP: ip}
	}

	rateLimit := getEnvBytes("rate", 0)
	if *rateStr != "" {
		if r, err := parseBytes(*rateStr); err == nil {
			rateLimit = r
		}
	}

	// 构造 Transport
	var transport http.RoundTripper
	{
		dialer := &net.Dialer{LocalAddr: localAddr}
		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if *ipv4Only {
					network = "tcp4"
				}
				if *ipv6Only {
					network = "tcp6"
				}
				return dialer.DialContext(ctx, network, addr)
			},
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for i := 1; i <= *workers; i++ {
		wg.Add(1)
		go worker(ctx, i, &wg, *url, *interval, *userAgent, rateLimit, transport)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	fmt.Println("程序已启动，按 Ctrl+C 停止...")
	<-sig

	fmt.Println("\n接收到停止信号，停止中...")
	cancel()
	wg.Wait()
	fmt.Println("所有 Worker 已安全停止")
}
