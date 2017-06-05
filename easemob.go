package easemob

import (
	"net/http"

	"github.com/go-resty/resty"
)

type EM struct {
	clientID     string
	clientSecret string
	baseURL      string
	token        string
}

func New(clientID, clientSecret, baseURL string) *EM {
	em := &EM{clientID, clientSecret, baseURL, ``}
	em.init()
	return em
}

func (em *EM) init() {
	resty.SetDebug(true)
	resty.AddRetryCondition(func(r *resty.Response) (bool, error) {
		return (r.StatusCode() == http.StatusTooManyRequests || r.StatusCode() == http.StatusServiceUnavailable), nil
	})
	resty.OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
		r.SetAuthToken(em.token)
		r.SetHeader("Accept", "application/json")
		r.SetHeader(`Content-Type`, `application/json`)
		r.SetResult(map[string]interface{}{})
		return nil
	})

	// 更新token
	go em.refreshToken()
}

func (em *EM) RegisterSignelUser(username, password string) (bool, error) {
	resp, err := em.excute(resty.R().SetBody(map[string]string{
		`username`: username,
		`password`: password,
	}), resty.MethodPost, em.url(`/users`))
	if err != nil {
		return false, err
	}
	return resp.StatusCode() == http.StatusOK, nil
}

func (em *EM) excute(request *resty.Request, method, url string) (*resty.Response, error) {
	resp, err := request.Execute(method, url)
	if err != nil {
		return resp, err
	}

	if resp.StatusCode() == http.StatusUnauthorized { // 需要更新token
		if em.refreshToken() {
			return em.excute(request, method, url)
		}
	}
	return resp, err
}

func (em *EM) refreshToken() bool {
	resp, _ := resty.New().SetDebug(true).R().
		SetBody(map[string]string{
			`grant_type`:    `client_credentials`,
			`client_id`:     em.clientID,
			`client_secret`: em.clientSecret,
		}).
		SetResult(map[string]interface{}{}).
		Post(em.url(`/token`))
	info := *resp.Result().(*map[string]interface{})
	if token, ok := info[`access_token`].(string); ok {
		em.token = token
		return true
	}
	return false
}

func (em *EM) url(endpoint string) string {
	return em.baseURL + endpoint
}
