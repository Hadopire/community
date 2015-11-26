/*
 * Nanocloud Community, a comprehensive platform to turn any application
 * into a cloud solution.
 *
 * Copyright (C) 2015 Nanocloud Software
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

var (
	apiUrl string
	client *http.Client
)

type Ocs struct {
	XMLName xml.Name `xml:"ocs"`
	Meta    OcsMeta
}
type OcsMeta struct {
	XMLName    xml.Name `xml:"meta"`
	Status     string   `xml:"status"`
	StatusCode int      `xml:"statuscode"`
	Message    string   `xml:"message"`
}

func init() {
	client = &http.Client{
		// this code should be only on debug mode
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

func Configure() {
	apiUrl = fmt.Sprintf("%s://%s/ocs/v1.php/cloud", conf.protocol, conf.hostname)
}

func Create(username, password string) ReturnMsg {
	_, err := ocsRequest("POST", apiUrl+"/users", url.Values{
		"userid":   {username},
		"password": {password},
	})
	if err != nil {
		return ReturnMsg{Method: "Add", Err: err.Error(), Plugin: "owncloud", Email: username}
	} else {
		return ReturnMsg{Method: "Add", Err: "", Plugin: "owncloud", Email: username}
	}
}

func Delete(username string) ReturnMsg {
	log.Println(username)
	_, err := ocsRequest("DELETE", apiUrl+"/users/"+username, nil)
	if err != nil {
		return ReturnMsg{Method: "Delete", Err: err.Error(), Plugin: "owncloud", Email: username}
	} else {
		return ReturnMsg{Method: "Delete", Err: "", Plugin: "owncloud", Email: username}
	}
}

// Allows to edit attributes related to a user.
// The key could be one of these values :
// email, display, password or quota.
// Only admins can edit the quota value.
func Edit(username string, key string, value string) ReturnMsg {
	log.Println(username, key, value)
	_, err := ocsRequest("PUT", apiUrl+"/users/"+username, url.Values{
		"key":   {key},
		"value": {value},
	})
	if err != nil {
		return ReturnMsg{Method: "Edit", Err: err.Error(), Plugin: "owncloud", Email: username}
	} else {
		return ReturnMsg{Method: "Edit", Err: "", Plugin: "owncloud", Email: username}
	}
}

func ocsRequest(method, url string, data url.Values) (Ocs, error) {
	var o Ocs
	// do a new request to the api
	req, err := http.NewRequest(method, url, strings.NewReader(data.Encode()))
	if err != nil {
		return o, err
	}
	req.SetBasicAuth(conf.adminLogin, conf.adminPassword)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rsp, err := client.Do(req)
	if err != nil {
		return o, err
	}
	// verify the http status code
	if rsp.StatusCode >= 300 {
		return o, errors.New(fmt.Sprintf("HTTP error: %s", rsp.Status))
	}
	// read the response and parse it
	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return o, err
	}
	log.Println(string(b))
	err = xml.Unmarshal(b, &o)
	if err != nil {
		return o, err
	}
	// 100 is the successful status code of owncloud
	if o.Meta.StatusCode != 100 {
		err = errors.New(fmt.Sprintf("Owncloud error %d: %s", o.Meta.StatusCode, o.Meta.Status))
	}
	return o, err
}
