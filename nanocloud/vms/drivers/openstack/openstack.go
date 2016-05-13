package openstack

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/Nanocloud/community/nanocloud/vms"
)

type hash map[string]interface{}

func init() {
	vms.Register("openstack", &driver{})
}

func login(address string, tenant string, username string, password string) (string, string, error) {

	req, err := json.Marshal(hash{
		"auth": hash{
			"tenantName": tenant,
			"passwordCredentials": hash{
				"username": username,
				"password": password,
			},
		},
	})
	if err != nil {
		return "", "", err
	}

	buff := bytes.NewBuffer(req)
	resp, err := http.Post("http://"+address+":5000/v2.0/tokens", "application/json", buff)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode == 401 {
		return "", "", errors.New("Invalid credentials")
	} else if resp.StatusCode != 200 {
		return "", "", errors.New("Openstack login failed, code: " + resp.Status)
	}

	var data struct {
		Access struct {
			Token struct {
				Tenant struct {
					Id string `json:"id"`
				} `json:"tenant"`
				Id string `json:"id"`
			} `json:"token"`
		} `json:"access"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", "", err
	}

	return data.Access.Token.Id, data.Access.Token.Tenant.Id, nil
}
