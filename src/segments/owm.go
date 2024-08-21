package segments

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"

	"github.com/jandedobbeleer/oh-my-posh/src/properties"
	"github.com/jandedobbeleer/oh-my-posh/src/runtime"
)

type Owm struct {
	props properties.Properties
	env   runtime.Environment

	Temperature int
	Weather     string
	URL         string
	units       string
	UnitIcon    string
}

const (
	// APIKey openweathermap api key
	APIKey properties.Property = "api_key"
	// Location openweathermap location
	Location properties.Property = "location"
	// Location openweathermap location
	Latitude properties.Property = "lat"
	// Location openweathermap location
	Longitude properties.Property = "lon"
	// Units openweathermap units
	Units properties.Property = "units"
	// CacheKeyResponse key used when caching the response
	CacheKeyResponse string = "owm_response"
	// CacheKeyURL key used when caching the url responsible for the response
	CacheKeyURL string = "owm_url"
	// Environmental variable to dynamically set the Open Map API key
	PoshOWMAPIKey string = "POSH_OWM_API_KEY"
	// Environmental variable to dynamically set the location string
	PoshOWMLocationKey string = "POSH_OWM_LOCATION"
	// Environmental variable to dynamically set the latitude
	PoshOWMLatKey string = "POSH_OWM_LAT"
	// Environmental variable to dynamically set the longitude
	PoshOWMLonKey string = "POSH_OWM_LON"
)

type weather struct {
	ShortDescription string `json:"main"`
	Description      string `json:"description"`
	TypeID           string `json:"icon"`
}
type temperature struct {
	Value float64 `json:"temp"`
}

type owmDataResponse struct {
	Data        []weather `json:"weather"`
	temperature `json:"main"`
}

func (d *Owm) Enabled() bool {
	err := d.setStatus()

	if err != nil {
		d.env.Error(err)
		return false
	}

	return true
}

func (d *Owm) Template() string {
	return " {{ .Weather }} ({{ .Temperature }}{{ .UnitIcon }}) "
}

func (d *Owm) getPropOrEnvVar(envKey, defaultValue string, propKeyOptions ...properties.Property) string {
	v := properties.OneOf(d.props, defaultValue, propKeyOptions...)
	if len(v) == 0 {
		v = d.env.Getenv(envKey)
	}
	return v
}

func (d *Owm) getResult() (*owmDataResponse, error) {
	cacheTimeout := d.props.GetInt(properties.CacheTimeout, properties.DefaultCacheTimeout)
	response := new(owmDataResponse)

	if cacheTimeout > 0 {
		val, found := d.env.Cache().Get(CacheKeyResponse)
		if found {
			err := json.Unmarshal([]byte(val), response)
			if err != nil {
				return nil, err
			}

			d.URL, _ = d.env.Cache().Get(CacheKeyURL)
			return response, nil
		}
	}

	apikey := d.getPropOrEnvVar(PoshOWMAPIKey, ".", APIKey, "apiKey")
	if len(apikey) == 0 {
		return nil, errors.New("no api key found")
	}

	units := d.props.GetString(Units, "standard")
	httpTimeout := d.props.GetInt(properties.HTTPTimeout, properties.DefaultHTTPTimeout)

	location := d.getPropOrEnvVar(PoshOWMLocationKey, "De Bilt,NL", Location)
	// location = url.QueryEscape(location)

	// Use different URLs depending on if a location or lat/lon were passed
	if len(location) > 0 {
		location = url.QueryEscape(location)
		d.URL = fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&units=%s&appid=%s", location, units, apikey)
	} else {
		lat := d.getPropOrEnvVar(PoshOWMLatKey, "0", Latitude)
		lat = url.QueryEscape(lat)
		lon := d.getPropOrEnvVar(PoshOWMLonKey, "0", Longitude)
		lon = url.QueryEscape(lon)
		d.URL = fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%s&lon=%s&units=%s&appid=%s", lat, lon, units, apikey)
	}

	body, err := d.env.HTTPRequest(d.URL, nil, httpTimeout)
	if err != nil {
		return new(owmDataResponse), err
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return new(owmDataResponse), err
	}

	if cacheTimeout > 0 {
		// persist new forecasts in cache
		d.env.Cache().Set(CacheKeyResponse, string(body), cacheTimeout)
		d.env.Cache().Set(CacheKeyURL, d.URL, cacheTimeout)
	}
	return response, nil
}

func (d *Owm) setStatus() error {
	units := d.props.GetString(Units, "standard")

	q, err := d.getResult()
	if err != nil {
		return err
	}

	if len(q.Data) == 0 {
		return errors.New("No data found")
	}

	id := q.Data[0].TypeID

	d.Temperature = int(math.Round(q.temperature.Value))
	icon := ""
	switch id {
	case "01n":
		icon = "\ue32b"
	case "01d":
		icon = "\ue30d"
	case "02n":
		icon = "\ue37e"
	case "02d":
		icon = "\ue302"
	case "03n":
		fallthrough
	case "03d":
		icon = "\ue33d"
	case "04n":
		fallthrough
	case "04d":
		icon = "\ue312"
	case "09n":
		fallthrough
	case "09d":
		icon = "\ue319"
	case "10n":
		icon = "\ue325"
	case "10d":
		icon = "\ue308"
	case "11n":
		icon = "\ue32a"
	case "11d":
		icon = "\ue30f"
	case "13n":
		fallthrough
	case "13d":
		icon = "\ue31a"
	case "50n":
		fallthrough
	case "50d":
		icon = "\ue313"
	}
	d.Weather = icon
	d.units = units
	d.UnitIcon = "\ue33e"
	switch d.units {
	case "imperial":
		d.UnitIcon = "°F" // \ue341"
	case "metric":
		d.UnitIcon = "°C" // \ue339"
	case "":
		fallthrough
	case "standard":
		d.UnitIcon = "°K" // <b>K</b>"
	}
	return nil
}

func (d *Owm) Init(props properties.Properties, env runtime.Environment) {
	d.props = props
	d.env = env
}
