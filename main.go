package main

import (
	"context"
	"fmt"
	api "go-weather-app/api"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"nhooyr.io/websocket"
)

type server struct {
	subscriberMessageBuffer int
	mux                     http.ServeMux
	subscribersMu           sync.Mutex
	subscribers             map[*subscriber]struct{}
}

type subscriber struct {
	msgs chan []byte
}

func NewServer() *server {
	s := &server{
		subscriberMessageBuffer: 10,
		subscribers:             make(map[*subscriber]struct{}),
	}
	s.mux.Handle("/", http.FileServer(http.Dir("./templates")))
	s.mux.HandleFunc("/ws", s.subscribeHandler)
	return s
}

func (s *server) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	err := s.subscribe(r.Context(), w, r)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (s *server) addSubscriber(subscriber *subscriber) {
	s.subscribersMu.Lock()
	s.subscribers[subscriber] = struct{}{}
	s.subscribersMu.Unlock()
	fmt.Println("Added subscriber", subscriber)
}

func (s *server) subscribe(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var c *websocket.Conn
	subscriber := &subscriber{
		msgs: make(chan []byte, s.subscriberMessageBuffer),
	}
	s.addSubscriber(subscriber)

	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return err
	}
	defer c.CloseNow()

	ctx = c.CloseRead(ctx)
	for {
		select {
		case msg := <-subscriber.msgs:
			ctx, cancel := context.WithTimeout(ctx, time.Second*5)
			defer cancel()
			err := c.Write(ctx, websocket.MessageText, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (cs *server) publishMsg(msg []byte) {
	cs.subscribersMu.Lock()
	defer cs.subscribersMu.Unlock()

	for s := range cs.subscribers {
		s.msgs <- msg
	}
}

func main() {
	fmt.Println("Starting monitor server on port 8000")
	envErr := godotenv.Load()
	if envErr != nil {
		log.Fatalf("Error loading .env file: %v", envErr)
	}
	s := NewServer()

	go func(srv *server) {
		for {
			weatherData, err := api.FetchWeather("Phnom Penh")
			if err != nil {
				fmt.Println(err)
				continue
			}

			// Convert wind speed to kilometers per hour
			windSpeedKmh := weatherData.Wind.Speed * 3.6
			windSpeedStr := strconv.FormatFloat(windSpeedKmh, 'f', 2, 64) + "km/h"

			// Convert humidity to percentage
			humidityStr := strconv.FormatFloat(float64(weatherData.Main.Humidity), 'f', -1, 64) + "%"

			// Convert visibility to kilometers
			visibilityKm := weatherData.Visibility / 1000
			visibilityStr := strconv.FormatFloat(float64(visibilityKm), 'f', -1, 64) + "km"

			timeStamp := time.Now().Format("January 2, 2006, 3:04 PM")
			msg := []byte(`
    <div hx-swap-oob="innerHTML:#update-timestamp">
    	<p>` + timeStamp + `</p>
    </div>
	<div hx-swap-oob="innerHTML:#weather-data-name">` + weatherData.Name + `</div>
	<div hx-swap-oob="innerHTML:#weather-data-temp">` + strconv.FormatFloat(weatherData.Main.Temp, 'f', -1, 64) + `°</div>
	<div hx-swap-oob="innerHTML:#weather-data-weather-main">` + weatherData.Weather[0].Description + `</div>
	<div hx-swap-oob="innerHTML:#weather-data-main-temp-max">` + strconv.FormatFloat(weatherData.Main.TempMax, 'f', -1, 64) + `</div>  
	<div hx-swap-oob="innerHTML:#weather-data-main-temp-min">` + strconv.FormatFloat(weatherData.Main.TempMin, 'f', -1, 64) + `</div>  
	<div hx-swap-oob="innerHTML:#weather-data-wind">` + windSpeedStr + `</div>
	<div hx-swap-oob="innerHTML:#weather-data-humidity">` + humidityStr + `</div>
	<div hx-swap-oob="innerHTML:#weather-data-visibility">` + visibilityStr + `</div>
	`)
			srv.publishMsg(msg)
			time.Sleep(3 * time.Second)
		}
	}(s)

	err := http.ListenAndServe(":8000", &s.mux)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
