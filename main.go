package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/googollee/go-socket.io"
)

// Attack is ...
type Attack struct {
	SrcLat     float32 `json:"srcLat"`
	SrcLng     float32 `json:"srcLong"`
	DstLat     float32 `json:"dstLat"`
	DstLong    float32 `json:"dstLong"`
	SrcIP      string  `json:"srcIP"`
	DstIP      string  `json:"dstIP"`
	AttackType string  `json:"attackType"`
}

// IPStackResponse is ...
type IPStackResponse struct {
	Lat float32 `json:"latitude"`
	Lng float32 `json:"longitude"`
}

func main() {
	// ip stack key
	key := os.Args[1]
	fmt.Println(key)
	c := make(chan Attack)

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

			url := fmt.Sprintf("http://api.ipstack.com/%s?access_key=%s&format=1", attack.SrcIP, key)
			resp, _ := http.Get(url)
			body, _ = ioutil.ReadAll(resp.Body)
			ipStackResponse := IPStackResponse{}
			json.Unmarshal(body, &ipStackResponse)

			attack.SrcLat = ipStackResponse.Lat
			attack.SrcLng = ipStackResponse.Lng

			url = fmt.Sprintf("http://api.ipstack.com/%s?access_key=%s&format=1", attack.DstIP, key)
			resp, _ = http.Get(url)
			body, _ = ioutil.ReadAll(resp.Body)
			ipStackResponse = IPStackResponse{}
			json.Unmarshal(body, &ipStackResponse)

			attack.DstLat = ipStackResponse.Lat
			attack.DstLong = ipStackResponse.Lng

			c <- attack

			fmt.Fprintf(w, "Post Success %v", attack)
		} else {
			fmt.Fprintf(w, "Only POST methods are supported.")
		}
	})

	http.Handle("/", http.FileServer(http.Dir("./public")))
	log.Println("Serving at localhost:3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
