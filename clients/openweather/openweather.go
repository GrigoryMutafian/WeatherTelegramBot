package openweather

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/flaticols/countrycodes"
)

type OpenWeatherClient struct {
	apiKey string
}

func New(apiKey string) *OpenWeatherClient {
	return &OpenWeatherClient{
		apiKey: apiKey,
	}
}

func (o OpenWeatherClient) Coordinates(city string) (Coordinates, string, error) {
	url := "http://api.openweathermap.org/geo/1.0/direct?q=%s&limit=5&appid=%s"
	resp, err := http.Get(fmt.Sprintf(url, city, o.apiKey))
	if err != nil {
		return Coordinates{}, "", fmt.Errorf("error get coodinates: %w", err)
	}

	if resp.StatusCode != 200 {
		return Coordinates{}, "", fmt.Errorf("error while getting coordinates. code:, %d", resp.StatusCode)
	}

	var coordinatesResponse []CoordinatesResponse

	err = json.NewDecoder(resp.Body).Decode(&coordinatesResponse)
	if err != nil {
		return Coordinates{}, "", fmt.Errorf("error while decoding body: %w", err)
	}

	if len(coordinatesResponse) == 0 {
		return Coordinates{}, "", fmt.Errorf("coordinates response is empty")
	}

	return Coordinates{
		Lat: coordinatesResponse[0].Lat,
		Lon: coordinatesResponse[0].Lon,
	}, coordinatesResponse[0].Country, nil
}

func (o OpenWeatherClient) CountryName(country string) string {
	if name, ok := countrycodes.Alpha2ToName(country); ok {
		country = name
	}
	return country
}

func (o OpenWeatherClient) Weather(lat, lon float64) (Weather, error) {
	url := "https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%s"
	resp, err := http.Get(fmt.Sprintf(url, lat, lon, o.apiKey))
	if err != nil {
		return Weather{}, fmt.Errorf("error get weather: %w", err)
	}

	if resp.StatusCode != 200 {
		return Weather{}, fmt.Errorf("error while getting weather. code: %d", resp.StatusCode)
	}

	var weatherResponse WeatherResponse

	err = json.NewDecoder(resp.Body).Decode(&weatherResponse)
	if err != nil {
		return Weather{}, fmt.Errorf("error while decoding response body: %w", err)
	}

	return Weather{
		Temp: weatherResponse.Main.Temp,
	}, nil
}
