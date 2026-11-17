package setting

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"
)

var customerVerifyClient = &http.Client{
	Timeout: 4 * time.Second,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

var customerVerifyURL atomic.Pointer[url.URL]

func loadCustomerVerifyURL(config *viper.Viper) {
	target, err := url.Parse(strings.TrimSpace(config.GetString("app.customer_verify_url")))
	if err != nil || target.Hostname() == "" || target.User != nil || (target.Scheme != "http" && target.Scheme != "https") {
		target = nil
	}
	customerVerifyURL.Store(target)
}

func CustomerVerifyURL() *url.URL {
	target := customerVerifyURL.Load()
	if target == nil {
		return nil
	}
	copy := *target
	return &copy
}

func VerifyCustomer(r *http.Request, id, name string) bool {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(name) == "" || len(id) > 256 || len(name) > 512 {
		return false
	}
	target := CustomerVerifyURL()
	if target == nil {
		return false
	}
	query := target.Query()
	query.Set("code", id)
	query.Set("username", name)
	target.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target.String(), nil)
	if err != nil {
		return false
	}
	response, err := customerVerifyClient.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1025))
	if err != nil || len(body) > 1024 {
		return false
	}
	var verified bool
	return json.Unmarshal(body, &verified) == nil && verified
}
