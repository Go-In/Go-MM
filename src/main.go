package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/go-redis/redis"
	"github.com/googollee/go-socket.io"
	"github.com/tkanos/gonfig"
)

// Attack is ...
type Attack struct {
	SrcLat         float32 `json:"srcLat"`
	SrcLng         float32 `json:"srcLong"`
	SrcCountryName string  `json:"srcCountryName"`
	DstLat         float32 `json:"dstLat"`
	DstLong        float32 `json:"dstLong"`
	DstCountryName string  `json:"dstCountryName"`
	SrcIP          string  `json:"src_ip"`
	DstIP          string  `json:"dest_ip"`
	AttackType     string  `json:"type"`
}

// IPStackResponse is ...
type IPStackResponse struct {
	Lat         float32 `json:"latitude"`
	Lng         float32 `json:"longitude"`
	CountryName string  `json:"country_name"`
}

// RedisConfig is ...
type RedisConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// Config is ...
type Config struct {
	RedisConfig RedisConfig `json:"redis"`
	IPStackKey  string      `json:"ipstack_key"`
	Host        string      `json:"host"`
	Port        string      `json:"port"`
}

var redisClient *redis.Client
var config Config

func getGeoFromIPStack(ip string) IPStackResponse {
	ipStackResponse := IPStackResponse{}
	if val, _ := redisClient.Get(ip).Result(); val != "" {
		log.Println("GET from redis")
		json.Unmarshal([]byte(val), &ipStackResponse)
	} else {
		log.Println("GET from ipstack 1234")
		url := fmt.Sprintf("http://localhost:5000/geo-ip?ip=%s", ip)
		resp, _ := http.Get(url)
		body, _ := ioutil.ReadAll(resp.Body)
		redisClient.Set(ip, string(body), 0)
		json.Unmarshal(body, &ipStackResponse)
	}
	return ipStackResponse
}

func main() {
	err := gonfig.GetConf("./src/config.json", &config)
	if err != nil {
		log.Fatal(err)
	}

	c := make(chan Attack)

	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.RedisConfig.Host, config.RedisConfig.Port),
		Password: config.RedisConfig.Password, // no password set
		DB:       config.RedisConfig.DB,       // use default DB
	})

	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	server.On("connection", func(so socketio.Socket) {
		log.Println("on connection")
		so.Join("suricata")
		so.On("disconnection", func() {
			log.Println("on disconnect")
		})
	})

	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})

	go func() {
		for {
			atk := <-c
			log.Println("receive from channel")
			log.Println(atk)
			server.BroadcastTo("suricata", "attacking", atk)
		}
	}()

	http.Handle("/socket.io/", server)

	http.HandleFunc("/attack", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, _ := ioutil.ReadAll(r.Body)
			attack := Attack{}
			json.Unmarshal(body, &attack)

			if strings.HasPrefix(attack.SrcIP, "158.108") && strings.HasPrefix(attack.DstIP, "158.108") {
				return
			}

			ipStackResponse := getGeoFromIPStack(attack.SrcIP)
			attack.SrcLat = ipStackResponse.Lat
			attack.SrcLng = ipStackResponse.Lng
			attack.SrcCountryName = ipStackResponse.CountryName

			ipStackResponse = getGeoFromIPStack(attack.DstIP)
			attack.DstLat = ipStackResponse.Lat
			attack.DstLong = ipStackResponse.Lng
			attack.DstCountryName = ipStackResponse.CountryName

			c <- attack

			fmt.Fprintf(w, "Post Success %v", attack)
		} else {
			fmt.Fprintf(w, "Only POST methods are supported.")
		}
	})

	http.Handle("/", http.FileServer(http.Dir("./src/public")))
	log.Println(fmt.Sprintf("Serving at %s:%s...", config.Host, config.Port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", config.Port), nil))
}
