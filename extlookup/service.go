package extlookup

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/OmniCar/autobot/config"
)

// Service handles the shared part of various Service implementations.
type Service struct {
	Name string
	Conf config.LookupConfig
}

// Configure takes a LookupConfig (which would probably come from a configuration file).
func (service *Service) Configure(cnf config.LookupConfig) error {
	service.Conf = cnf
	return nil
}

// makeReq performs the request against the license plate service and returns
// the HTTP response as a byte slice.
func (service *Service) makeReq(reqURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return []byte{}, err
	}
	req.Header = map[string][]string{
		"Content-Type": {"application/json"},
	}
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	if res.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("service responded with status code %d", res.StatusCode)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, err
	}
	return body, nil
}
