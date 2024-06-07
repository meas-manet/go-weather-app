package main

import (
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	api "go-weather-app/api"

	"github.com/joho/godotenv"
)

type Server struct {
	mu   sync.Mutex
	data api.WeatherResponse
}

func (s *Server) updateWeatherData(city string) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		data, err := api.FetchWeather(city)
		if err != nil {
			log.Printf("Error fetching weather data: %v", err)
			continue
		}
		s.mu.Lock()
		log.Printf("fetching weather data...... ")
		s.data = data
		s.mu.Unlock()
	}
}

func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	data := s.data
	s.mu.Unlock()

	templ := template.Must(template.ParseFiles("templates/index.html"))
	templ.Execute(w, data)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	server := &Server{}

	go server.updateWeatherData("Phnom Penh")

	http.HandleFunc("/", server.handler)

	log.Println("Starting server on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
