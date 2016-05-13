package openstack

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/Nanocloud/community/nanocloud/connectors/db"
	"github.com/Nanocloud/community/nanocloud/vms"
)

type machine struct {
	id       string
	name     string
	address  string
	tenant   string
	username string
	password string
	ip       net.IP
	mType    vms.MachineType
}

type DriverNetwork struct {
	Addr string `json:"addr"`
	Type string `json:"OS-EXT-IPS:type"`
}

func (m *machine) Id() string {
	return m.id
}

func (m *machine) Platform() string {
	return "not implemented"
}

func (m *machine) Name() (string, error) {
	return m.name, nil
}

func (m *machine) Status() (vms.MachineStatus, error) {

	token, tenantId, err := login(m.address, m.tenant, m.username, m.password)
	if err != nil {
		return vms.StatusUnknown, err
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, "http://"+m.address+":8774/v2/"+tenantId+"/servers/"+m.id, nil)
	if err != nil {
		return vms.StatusUnknown, err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return vms.StatusUnknown, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return vms.StatusUnknown, err
	}

	var data struct {
		Server struct {
			Addresses struct {
				DriverNetwork interface{} `json:"driverNetwork"`
			} `json:"addresses"`
			Id        string `json:"id"`
			Name      string `json:"name"`
			VmState   string `json:"OS-EXT-STS:vm_state"`
			TaskState string `json:"OS-EXT-STS:task_state"`
		} `json:"server"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return vms.StatusUnknown, err
	}

	m.id = data.Server.Id
	m.name = data.Server.Name
	m.ip = nil

	driverNetwork, ok := data.Server.Addresses.DriverNetwork.([]interface{})
	if ok {
		for _, address := range driverNetwork {
			addr := address.(map[string]interface{})
			if addr["OS-EXT-IPS:type"].(string) == "floating" {
				m.ip = net.ParseIP(addr["addr"].(string))
			}
		}
	}

	switch data.Server.VmState {
	case "active":
		if data.Server.TaskState == "powering-off" {
			return vms.StatusStopping, nil
		} else if data.Server.TaskState == "deleting" {
			return vms.StatusTerminated, nil
		}
		return vms.StatusUp, nil
	case "stopped":
		if data.Server.TaskState == "powering-on" {
			return vms.StatusBooting, nil
		}
		return vms.StatusDown, nil
	case "building":
		return vms.StatusBooting, nil
	default:
		return vms.StatusUnknown, nil
	}
}

func (m *machine) IP() (net.IP, error) {
	return m.ip, nil
}

func (m *machine) Type() (vms.MachineType, error) {

	return m.mType, nil
}

func (m *machine) Progress() (uint8, error) {
	return 100, nil
}

func (m *machine) Action(b interface{}) error {

	token, tenantId, err := login(m.address, m.tenant, m.username, m.password)
	if err != nil {
		return err
	}

	reqBody, err := json.Marshal(b)
	if err != nil {
		return err
	}

	buff := bytes.NewBuffer(reqBody)
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, "http://"+m.address+":8774/v2/"+tenantId+"/servers/"+m.id+"/action", buff)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		fmt.Println(string(body))
		return errors.New(resp.Status)
	}
	return nil
}

func (m *machine) Start() error {

	err := m.Action(hash{
		"os-start": nil,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *machine) Stop() error {

	err := m.Action(hash{
		"os-stop": nil,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *machine) Terminate() error {

	token, tenantId, err := login(m.address, m.tenant, m.username, m.password)
	if err != nil {
		return err
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodDelete, "http://"+m.address+":8774/v2/"+tenantId+"/servers/"+m.id, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		fmt.Println(string(body))
		return errors.New(resp.Status)
	}

	res, err := db.Exec("DELETE FROM machines WHERE id = $1::varchar", m.id)
	if err != nil {
		return err
	}
	deleted, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return errors.New("machine entry not found")
	}
	return nil
}

func (m *machine) Credentials() (string, string, error) {

	var password string
	rows, err := db.Query(
		`SELECT password
			FROM machines WHERE id = $1::varchar`, m.id)
	if err != nil {
		return "", "", err
	}
	for rows.Next() {
		rows.Scan(
			&password,
		)
	}
	err = rows.Err()
	if err != nil {
		return "", "", err
	}
	return "Administrator", password, nil
}
