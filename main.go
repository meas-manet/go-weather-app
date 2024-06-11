package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"nhooyr.io/websocket"

	fetchweather "go-weather-app/api"
)

type server struct {
	subscriberMessageBuffer int
	mux                     *http.ServeMux
	subscribersMu           sync.Mutex
	subscribers             map[*subscriber]struct{}
	city                    string
}

type subscriber struct {
	msgs chan []byte
}

// NewServer creates a new server instance with initialized values.
func NewServer() *server {
	s := &server{
		subscriberMessageBuffer: 10,
		mux:                     http.NewServeMux(),
		subscribers:             make(map[*subscriber]struct{}),
		city:                    "Phnom Penh", // Default city
	}
	s.mux.Handle("/", http.FileServer(http.Dir("./templates")))
	s.mux.HandleFunc("/ws", s.subscribeHandler)
	s.mux.HandleFunc("/update-city", s.updateCityHandler)
	return s
}

// subscribeHandler handles WebSocket subscription requests.
func (s *server) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.subscribe(r.Context(), w, r); err != nil {
		log.Println("Subscription error:", err)
	}
}

// addSubscriber adds a new subscriber to the server.
func (s *server) addSubscriber(sub *subscriber) {
	s.subscribersMu.Lock()
	s.subscribers[sub] = struct{}{}
	s.subscribersMu.Unlock()
	log.Println("Added subscriber")
}

// removeSubscriber removes a subscriber from the server.
func (s *server) removeSubscriber(sub *subscriber) {
	s.subscribersMu.Lock()
	delete(s.subscribers, sub)
	s.subscribersMu.Unlock()
	log.Println("Removed subscriber")
}

// subscribe handles the WebSocket connection and message sending for a subscriber.
func (s *server) subscribe(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return fmt.Errorf("failed to accept WebSocket connection: %w", err)
	}
	defer c.Close(websocket.StatusNormalClosure, "Closing")

	sub := &subscriber{
		msgs: make(chan []byte, s.subscriberMessageBuffer),
	}
	s.addSubscriber(sub)
	defer s.removeSubscriber(sub)

	go s.sendWeatherUpdates(ctx, sub)

	for {
		select {
		case msg := <-sub.msgs:
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := c.Write(ctx, websocket.MessageText, msg); err != nil {
				return fmt.Errorf("failed to write message: %w", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// sendWeatherUpdates sends weather updates to the subscriber based on the current city.
func (s *server) sendWeatherUpdates(ctx context.Context, sub *subscriber) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			weatherData, err := fetchweather.FetchWeather(s.city)
			if err != nil {
				log.Println("Error fetching weather data:", err)
				time.Sleep(5 * time.Second)
				continue
			}
			msg := formatWeatherData(weatherData)
			sub.msgs <- msg
			time.Sleep(3 * time.Second)
		}
	}
}

// updateCityHandler handles requests to update the city.
func (s *server) updateCityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	city := r.FormValue("city")
	if city == "" {
		http.Error(w, "City is required", http.StatusBadRequest)
		return
	}

	s.city = city
	log.Println("City updated to", city)

	// Notify subscribers of the new city
	s.subscribersMu.Lock()
	for sub := range s.subscribers {
		close(sub.msgs)
		sub.msgs = make(chan []byte, s.subscriberMessageBuffer)
		go s.sendWeatherUpdates(context.Background(), sub)
	}
	s.subscribersMu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// formatWeatherData formats weather data into an HTML message.
func formatWeatherData(data fetchweather.WeatherResponse) []byte {
	windSpeedKmh := data.Wind.Speed * 3.6
	windSpeedStr := fmt.Sprintf("%.2f km/h", windSpeedKmh)

	humidityStr := fmt.Sprintf("%.0f%%", float64(data.Main.Humidity))

	visibilityKm := data.Visibility / 1000
	visibilityStr := fmt.Sprintf("%.0f km", float64(visibilityKm))

	timeStamp := time.Now().Format("January 2, 2006, 3:04 PM")

	return []byte(fmt.Sprintf(`
    <div hx-swap-oob="innerHTML:#update-timestamp">
    	<p>%s</p>
    </div>
	<div hx-swap-oob="innerHTML:#weather-data-name">%s</div>
	<div hx-swap-oob="innerHTML:#weather-data-temp">%.0f°</div>
	<div hx-swap-oob="innerHTML:#weather-data-weather-main">%s</div>
	<div hx-swap-oob="innerHTML:#weather-data-main-temp-max">%.0f</div>  
	<div hx-swap-oob="innerHTML:#weather-data-main-temp-min">%.0f</div>  
	<div hx-swap-oob="innerHTML:#weather-data-wind">%s</div>
	<div hx-swap-oob="innerHTML:#weather-data-humidity">%s</div>
	<div hx-swap-oob="innerHTML:#weather-data-visibility">%s</div>
	`,
		timeStamp,
		data.Name,
		data.Main.Temp,
		data.Weather[0].Description,
		data.Main.TempMax,
		data.Main.TempMin,
		windSpeedStr,
		humidityStr,
		visibilityStr,
	))
}

func main() {
	log.Println("Starting monitor server on port 8000")
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	s := NewServer()

	if err := http.ListenAndServe(":8000", s.mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
