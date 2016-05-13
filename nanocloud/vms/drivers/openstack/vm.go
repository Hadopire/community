package openstack

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/Nanocloud/community/nanocloud/connectors/db"
	"github.com/Nanocloud/community/nanocloud/plaza"
	"github.com/Nanocloud/community/nanocloud/provisioner"
	"github.com/Nanocloud/community/nanocloud/vms"
)

type vm struct {
	address  string
	tenant   string
	username string
	password string
	image    string
}

func (v *vm) Machines() ([]vms.Machine, error) {

	token, tenantId, err := login(v.address, v.tenant, v.username, v.password)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, "http://"+v.address+":8774/v2/"+tenantId+"/servers/detail", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Servers []struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"servers"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	machines := make([]vms.Machine, 0)
	for _, instance := range data.Servers {
		machines = append(machines, &machine{
			id:       instance.Id,
			name:     instance.Name,
			address:  v.address,
			tenant:   v.tenant,
			username: v.username,
			password: v.password,
		})
	}
	return machines, nil
}

func (v *vm) Machine(id string) (vms.Machine, error) {

	token, tenantId, err := login(v.address, v.tenant, v.username, v.password)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, "http://"+v.address+":8774/v2/"+tenantId+"/servers/"+id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Server struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"server"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return &machine{
		id:       data.Server.Id,
		name:     data.Server.Name,
		address:  v.address,
		tenant:   v.tenant,
		username: v.username,
		password: v.password,
	}, nil
}

func (v *vm) Create(attr vms.MachineAttributes) (vms.Machine, error) {

	token, tenantId, err := login(v.address, v.tenant, v.username, v.password)
	if err != nil {
		return nil, err
	}

	if attr.Type == nil {
		return nil, errors.New("No machine type specified")
	}

	t, ok := attr.Type.(*machineType)
	if !ok {
		return nil, errors.New("VM Type not supported")
	}

	client := &http.Client{}

	reqBody, err := json.Marshal(hash{
		"server": hash{
			"imageRef":  t.image,
			"flavorRef": t.flavorHref,
			"name":      attr.Name,
			"key_name":  "NanocloudKey",
			"user_data": base64.StdEncoding.EncodeToString([]byte("#ps1_sysnative\r\n$domain = hostname\r\n$user = [adsi]\"WinNT://$domain/Administrator\"\r\n$user.changePassword(\"\", \"Nanocloud123+\")\r\nInvoke-WebRequest https://s3-eu-west-1.amazonaws.com/nanocloud/plaza.exe -OutFile C:\\plaza.exe\r\nC:\\plaza.exe\r\nNew-NetFirewallRule -Protocol TCP -LocalPort 9090 -Direction Inbound -Action Allow -DisplayName PLAZA")),
		},
	})
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(reqBody)
	req, err := http.NewRequest(http.MethodPost, "http://"+v.address+":8774/v2/"+tenantId+"/servers", buff)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Server struct {
			Id string `json:"id"`
		} `json:"server"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`INSERT INTO machines
		(id, password)
		VALUES( $1::varchar, $2::varchar )`,
		data.Server.Id, attr.Password)
	if err != nil {
		return nil, err
	}
	rows.Close()

	mac, err := v.Machine(data.Server.Id)
	if err != nil {
		return nil, err
	}
	mac.Credentials()
	p := provisioner.New(plaza.Provision(mac))
	go p.Run()
	p.AddOutput(os.Stdout)

	return mac, nil
}

func (v *vm) Types() ([]vms.MachineType, error) {

	token, tenantId, err := login(v.address, v.tenant, v.username, v.password)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, "http://"+v.address+":8774/v2/"+tenantId+"/flavors", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		Flavors []struct {
			Links []struct {
				Href string `json:"href"`
			} `json:"links"`
			Name string `json:"name"`
		} `json:"flavors"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	types := make([]vms.MachineType, 0)

	for _, flavor := range data.Flavors {
		types = append(types, &machineType{
			flavor:     flavor.Name,
			flavorHref: flavor.Links[0].Href,
			image:      v.image,
		})
	}

	return types, nil
}

func (v *vm) Type(id string) (vms.MachineType, error) {

	types, err := v.Types()
	if err != nil {
		return nil, err
	}

	for _, t := range types {
		if t.GetID() == id {
			return t, nil
		}
	}

	return nil, errors.New("Type does not exists")
}
