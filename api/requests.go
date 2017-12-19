package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

const (
	// PartnerID is the var name in API calls
	PartnerID = "partner_id"
	// ServerToken is the var name in API calls
	ServerToken = "server_token"
)

// NambaTaxiAPI is a main request struct representing Namba Partners API
type NambaTaxiAPI struct {
	partnerID   string
	serverToken string
	url         string
	version     string
}

// Fares struct represents the list of fares of Namba Taxi
// fare ID is used when creating an order
type Fares struct {
	Fare []struct {
		Flagfall          float64 `json:"flagfall"`
		FreeWaiting       float64 `json:"free_waiting"`
		FullDescription   string  `json:"full_description"`
		IncludeKilometers int     `json:"include_kilometers"`
		ID                int     `json:"id"`
		CostPerKilometer  float64 `json:"cost_per_kilometer"`
		Name              string  `json:"name"`
	} `json:"fares"`
}

// PaymentMethods represents the list of payment options user can use for a ride
// In partners API there is no option to change payment option,
// so all requests are made with "cash" payment method,
// but this can be changed in future
type PaymentMethods struct {
	PaymentMethod []struct {
		PaymentMethodID int    `json:"payment_method_id"`
		Description     string `json:"description"`
	} `json:"payment_methods"`
}

// RequestOptions represents the list of options for the order (like
// "have a cat" or "drunk client"). Currently not used
type RequestOptions struct {
	RequestOption []struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	} `json:"request_options"`
}

// Order represents an order structure which returns after an order has been created
// or we ask API about the particular order. When a driver accepts an order
// than the "Driver" substruct is populated
type Order struct {
	OrderID  int    `json:"order_id"`
	Message  string `json:"message"`
	Status   string `json:"status"`
	TripCost string `json:"trip_cost"`
	Driver   struct {
		Name         string  `json:"name"`
		PhoneNumber  string  `json:"phone_number"`
		CabNumber    string  `json:"cab_number"`
		LicensePlate string  `json:"license_plate"`
		Make         string  `json:"make"`
		Lat          float64 `json:"lat"`
		Lon          float64 `json:"lon"`
	} `json:"driver"`
}

// Cancel struct returns when we cancel an order
type Cancel struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// NearestDrivers return when we call nearest_drivers API
// "Drivers" is actual fiels, others are for errors and info
type NearestDrivers struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Drivers int    `json:"drivers"`
}

// NewNambaTaxiApi is the NambaTaxiAPI structure constructor
func NewNambaTaxiApi(partnerID string, serverToken string, url string, version string) NambaTaxiAPI {
	return NambaTaxiAPI{partnerID, serverToken, url, version}
}

// GetNearestDrivers returns the number of free cabs near the given address.
// The address is specified as real address (like "green street, 6 building"),
// not coordinates.
func (api *NambaTaxiAPI) GetNearestDrivers(address string) (NearestDrivers, error) {
	postParams := map[string][]string{
		"address": {address},
	}
	structure := NearestDrivers{}
	err := api.makePostRequestAndMapStructure(&structure, "nearest_drivers", postParams)
	if err != nil {
		return NearestDrivers{}, err
	}
	return structure, nil
}

// GetFares return the list of all current fares with full information
// This info can be shown to the users as well.
func (api *NambaTaxiAPI) GetFares() (Fares, error) {
	structure := Fares{}
	err := api.makePostRequestAndMapStructure(&structure, "fares", make(map[string][]string))
	if err != nil {
		return Fares{}, err
	}
	return structure, nil
}

// GetPaymentMethods return the list of active payment methods,
// but in this bot we use only "cash" option
func (api *NambaTaxiAPI) GetPaymentMethods() (PaymentMethods, error) {
	structure := PaymentMethods{}
	err := api.makePostRequestAndMapStructure(&structure, "payment-methods", make(map[string][]string))
	if err != nil {
		return PaymentMethods{}, err
	}
	return structure, nil
}

// GetRequestOptions return the list of options user can set to the order.
// For now we do not use this feature in bot
func (api *NambaTaxiAPI) GetRequestOptions() (RequestOptions, error) {
	structure := RequestOptions{}
	err := api.makePostRequestAndMapStructure(&structure, "request-options", make(map[string][]string))
	if err != nil {
		return RequestOptions{}, err
	}
	return structure, nil
}

// MakeOrder is for order creation. Some code examples can be found in tests
func (api *NambaTaxiAPI) MakeOrder(orderOptions map[string][]string) (Order, error) {
	structure := Order{}
	err := api.makePostRequestAndMapStructure(&structure, "requests", orderOptions)
	if err != nil {
		return Order{}, err
	}
	return structure, nil
}

// GetOrder is used to check the order status by its ID. This is useful
// to inform a user about the order status changing
func (api *NambaTaxiAPI) GetOrder(id int) (Order, error) {
	structure := Order{}
	err := api.makePostRequestAndMapStructure(&structure, "requests/"+strconv.Itoa(id), make(map[string][]string))
	if err != nil {
		return Order{}, err
	}
	return structure, nil
}

// CancelOrder is for oder canceling. An order can be canceled
// only if it has driver free status. When a driver had accepted an order
// it can not be canceled
func (api *NambaTaxiAPI) CancelOrder(id int) (Cancel, error) {
	structure := Cancel{}
	err := api.makePostRequestAndMapStructure(&structure, "cancel_order/"+strconv.Itoa(id), make(map[string][]string))
	if err != nil {
		return Cancel{}, err
	}
	return structure, nil
}

// makePostRequestAndMapStructure is an internal helper func that is used to
// make a post request and map a giver struct to a returned json
func (api *NambaTaxiAPI) makePostRequestAndMapStructure(structure interface{}, uri string, postParams map[string][]string) error {
	jsonData, err := api.makePostRequest(uri, postParams)
	if err != nil {
		return err
	}
	err = json.Unmarshal(jsonData, structure)
	if err != nil {
		return err
	}
	return nil
}

// makePostRequest is a helper internal func that is used to
// form all post params. We just give it what we have, and get
// the whole list with system post params as well
func (api *NambaTaxiAPI) makePostRequest(uri string, postParams map[string][]string) ([]byte, error) {
	var values url.Values = map[string][]string{
		PartnerID:   {api.partnerID},
		ServerToken: {api.serverToken},
	}

	for key, value := range postParams {
		values[key] = value
	}

	resp, err := http.PostForm(api.getAPIURL(uri), values)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	log.Printf("%s", string(body))
	return body, nil
}

// getAPIURL returns API url with version and URI
func (api *NambaTaxiAPI) getAPIURL(uri string) string {
	urlString := fmt.Sprintf("%s/%s/%s/", api.url, api.version, uri)
	log.Printf("API URL is: %v", urlString)
	return urlString
}