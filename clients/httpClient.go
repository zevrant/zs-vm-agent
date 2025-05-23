package clients

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const FORM_URL_ENCODED = "application/x-www-form-urlencoded"

type Client struct {
	hostURL               string
	httpClient            *http.Client
	token                 string
	auth                  AuthStruct
	enableTLSVerification bool
	logger                *logrus.Logger
}

type AuthStruct struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// NewClient -
func NewClient(host string, username string, password string, verifyTls bool, aLogger *logrus.Logger) *Client {
	if host == "" {
		panic("Host Not Provided!!!!")
	}

	c := Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		hostURL:    host,
		auth: AuthStruct{
			Username: username,
			Password: password,
		},
		enableTLSVerification: verifyTls,
		logger:                aLogger,
	}

	return &c
}

func (c *Client) doRequest(req *http.Request, contentType string) ([]byte, error) {
	return c.doRequestWithResponseStatus(req, http.StatusOK, contentType)
}

func (c *Client) doRequestWithResponseStatus(req *http.Request, expectedResponseStatus int, contentType string) ([]byte, error) {
	c.logger.Debug(fmt.Sprintf("Making %s request to %s", req.Method, req.URL))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)
	if !c.enableTLSVerification {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	c.logger.Debug(fmt.Sprintf("status code was %d for url %s", res.StatusCode, req.URL.Path))
	if res.StatusCode != expectedResponseStatus {
		c.logger.Error(fmt.Sprintf("statusCode: %d, status:%s, body: %s", res.StatusCode, res.Status, body))
		return body, fmt.Errorf("status: %s, body: %s", res.Status, body)
	}

	return body, err
}
