package fetchweather

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

// WeatherResponse represents the structure of the weather data returned by the API.
type WeatherResponse struct {
	Coord struct {
		Lon float64 `json:"lon"`
		Lat float64 `json:"lat"`
	} `json:"coord"`
	Weather []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"weather"`
	Base string `json:"base"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		TempMin   float64 `json:"temp_min"`
		TempMax   float64 `json:"temp_max"`
		Pressure  int     `json:"pressure"`
		Humidity  int     `json:"humidity"`
		SeaLevel  int     `json:"sea_level"`
		GrndLevel int     `json:"grnd_level"`
	} `json:"main"`
	Visibility int `json:"visibility"`
	Wind       struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
		Gust  float64 `json:"gust"`
	} `json:"wind"`
	Rain struct {
		OneHour float64 `json:"1h"`
	} `json:"rain"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Dt  int64 `json:"dt"`
	Sys struct {
		Type    int    `json:"type"`
		ID      int    `json:"id"`
		Country string `json:"country"`
		Sunrise int64  `json:"sunrise"`
		Sunset  int64  `json:"sunset"`
	} `json:"sys"`
	Timezone int    `json:"timezone"`
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Cod      int    `json:"cod"`
}

// FetchWeather fetches weather data for a specified city using the API key from environment variables.
func FetchWeather(city string) (WeatherResponse, error) {
	apiKey, err := getAPIKey()
	if err != nil {
		return WeatherResponse{}, err
	}

	url := buildWeatherAPIURL(city, apiKey)
	weatherResponse, err := getWeatherData(url)
	if err != nil {
		return WeatherResponse{}, err
	}

	return weatherResponse, nil
}

// getAPIKey retrieves the API key from the environment variables.
func getAPIKey() (string, error) {
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		return "", errors.New("API_KEY environment variable is not set")
	}
	return apiKey, nil
}

// buildWeatherAPIURL constructs the URL for fetching weather data.
func buildWeatherAPIURL(city, apiKey string) string {
	return fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric", city, apiKey)
}

// getWeatherData sends a request to the provided URL and parses the response into a WeatherResponse.
func getWeatherData(url string) (WeatherResponse, error) {
	resp, err := http.Get(url)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("error fetching weather data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WeatherResponse{}, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherResponse{}, fmt.Errorf("error reading response body: %v", err)
	}

	var weatherResponse WeatherResponse
	if err := json.Unmarshal(body, &weatherResponse); err != nil {
		return WeatherResponse{}, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return weatherResponse, nil
}
