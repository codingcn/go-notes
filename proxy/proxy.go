package main

import (
	"github.com/PuerkitoBio/goquery"
	"log"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"net/url"
	"net/http"
	"encoding/json"
	"strconv"
	"math/rand"
	"time"
	"strings"
)

const (
	PAGE int = 40
)

var (
	xici        string = "http://www.xicidaili.com/nn/"
	checkUrl    string = "https://www.baidu.com"
	pool        *redis.Pool
	redisServer = "127.0.0.1:6379"
)

func main() {
	for i := 0; i < 100; i++ {
		go func() {
			for {
				checkAvailableIp()
			}
		}()
	}
	//getIp("local")
	getIp("http://180.118.86.44:9000")

}
func newPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3000,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr) },
	}
}
func getIp(ip string) {
	var count int

	for i := 1; i <= PAGE; i++ {
		response := getRep(xici+strconv.Itoa(i), ip)
		if (response.StatusCode == 200) {
			// 这是一个可用ip，我们可以存起来
			saveAvailableIpRedis(ip)

			dom, err := goquery.NewDocumentFromReader(response.Body)
			if err != nil {
				log.Fatalf("失败原因", response.StatusCode)
			}
			dom.Find("#ip_list tbody tr").Each(func(i int, context *goquery.Selection) {
				ipInfo := make(map[string][]string)
				//地址
				ip := context.Find("td").Eq(1).Text()
				//端口
				port := context.Find("td").Eq(2).Text()
				//地址
				address := context.Find("td").Eq(3).Find("a").Text()
				//匿名
				anonymous := context.Find("td").Eq(4).Text()
				//协议
				protocol := context.Find("td").Eq(5).Text()
				//存活时间
				survivalTime := context.Find("td").Eq(8).Text()
				//验证时间
				checkTime := context.Find("td").Eq(9).Text()
				ipInfo[ip] = append(ipInfo[ip], ip, port, address, anonymous, protocol, survivalTime, checkTime)
				hBody, _ := json.Marshal(ipInfo[ip])

				//存入redis
				saveMixIpRedis(ip+":"+port, string(hBody))
				fmt.Println(ipInfo)
				count++
			})
		}
	}

}

/**
* 返回response
*/
func getRep(urls string, ip string) *http.Response {

	request, _ := http.NewRequest("GET", urls, nil)
	//随机返回User-Agent 信息
	request.Header.Set("User-Agent", getAgent())
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	request.Header.Set("Connection", "keep-alive")
	proxy, err := url.Parse(ip)
	//设置超时时间
	timeout := time.Duration(20 * time.Second)
	client := &http.Client{}
	if ip != "local" {
		client = &http.Client{
			Transport: &http.Transport{
				//TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				Proxy: http.ProxyURL(proxy),
			},
			Timeout: timeout,
		}
	}

	response, err := client.Do(request)
	if err != nil || response.StatusCode != 200 {
		fmt.Printf("代理ip不可用 %s\n", err)
		ip := returnIp()
		if ip != "" {
			fmt.Printf("切换ip %s\n", ip)
			getIp(ip)
		} else {
			log.Fatalf("Redis无可用ip")
		}
	}

	return response
}

/**
* 随机返回一个User-Agent
*/
func getAgent() string {
	agent := [...]string{
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:50.0) Gecko/20100101 Firefox/50.0",
		"Opera/9.80 (Macintosh; Intel Mac OS X 10.6.8; U; en) Presto/2.8.131 Version/11.11",
		"Opera/9.80 (Windows NT 6.1; U; en) Presto/2.8.131 Version/11.11",
		"Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 5.1; 360SE)",
		"Mozilla/5.0 (Windows NT 6.1; rv:2.0.1) Gecko/20100101 Firefox/4.0.1",
		"Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 5.1; The World)",
		"User-Agent,Mozilla/5.0 (Macintosh; U; Intel Mac OS X 10_6_8; en-us) AppleWebKit/534.50 (KHTML, like Gecko) Version/5.1 Safari/534.50",
		"User-Agent, Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 5.1; Maxthon 2.0)",
		"User-Agent,Mozilla/5.0 (Windows; U; Windows NT 6.1; en-us) AppleWebKit/534.50 (KHTML, like Gecko) Version/5.1 Safari/534.50",
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	len := len(agent)
	return agent[r.Intn(len)]
}

func checkAvailableIp() {
	ip := returnIp()
	if ip != "" {
		fmt.Printf("验证ip %s\n", ip)
	} else {
		log.Fatalf("Redis无可用ip")
	}
	request, _ := http.NewRequest("GET", checkUrl, nil)
	//随机返回User-Agent 信息
	request.Header.Set("User-Agent", getAgent())
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	request.Header.Set("Connection", "keep-alive")
	fmt.Printf("ipppp1 %s\n", ip)
	proxy, err := url.Parse(ip)
	fmt.Printf("ipppp2 %s\n", ip)
	//设置超时时间
	timeout := time.Duration(20 * time.Second)
	client := &http.Client{}
	if ip != "local" {
		client = &http.Client{
			Transport: &http.Transport{
				//TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				Proxy: http.ProxyURL(proxy),
			},
			Timeout: timeout,
		}
	}

	response, err := client.Do(request)
	if err != nil || response.StatusCode != 200 {
		fmt.Printf("代理ip验证不可用 %s\n", err)
		checkAvailableIp()
	} else {
		fmt.Printf("恭喜你，IP可用 %s\n", ip)
		saveAvailableIpRedis(ip)
	}
}
func saveAvailableIpRedis(ip string) {
	pool = newPool(redisServer)
	conn := pool.Get()
	u, err := url.Parse(ip)
	if err != nil {
		log.Fatalf("ip parse err:%s , %s", ip, err)
	}
	// 可用的IP代理池，键值对的方式存入hash
	_, err = conn.Do("HSET", "AVA_IP_POOL", u.Hostname(), string(ip))
	if err != nil {
		log.Fatalf("err:%s", err)
	}
}
func saveMixIpRedis(ip string, hBody string) {
	pool = newPool(redisServer)
	conn := pool.Get()
	defer conn.Close()
	//键值对的方式存入hash
	conn.Do("HSET", "MIX_IP_POOL", ip, string(hBody))
	//将ip:port 存入set  方便返回随机的ip
	conn.Do("SADD", "MIX_IP_POOL_KEY", ip)
}

/**
* 随机返回一个IP
*/
func returnIp() string {
	pool = newPool(redisServer)
	conn := pool.Get()
	key, err := redis.String(conn.Do("SPOP", "MIX_IP_POOL_KEY"))
	if err != nil {
		panic(err)
	}
	if key == "" {
		return ""
	}
	res, err := redis.String(conn.Do("HGET", "MIX_IP_POOL", key))
	if err != nil {
		panic(err)
	}
	res = strings.TrimLeft(res, "[")
	res = strings.TrimRight(res, "]")

	array := strings.Split(res, ",")

	for i := 0; i < len(array); i++ {
		array[i] = strings.Trim(array[i], "\"")
	}
	host := strings.ToLower(array[4]) + "://" + array[0] + ":" + array[1]
	return host

}
