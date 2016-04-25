/*
 * Nanocloud Community, a comprehensive platform to turn any application
 * into a cloud solution.
 *
 * Copyright (C) 2016 Nanocloud Software
 *
 * This file is part of Nanocloud community.
 *
 * Nanocloud community is free software; you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * Nanocloud community is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package manual

import (
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/Nanocloud/community/nanocloud/connectors/db"
	"github.com/Nanocloud/community/nanocloud/vms"
	"github.com/labstack/gommon/log"
)

type machine struct {
	id        string
	server    string
	plazaport string
	user      string
	password  string
}

func (m *machine) Status() (vms.MachineStatus, error) {
	resp, err := http.Get("http://" + m.server + ":" + m.plazaport + "/checkrds")
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Error(err)
		return vms.StatusUnknown, nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return vms.StatusUnknown, nil
	}

	if strings.Contains(string(b), "Running") {
		return vms.StatusUp, nil
	}
	return vms.StatusDown, nil

}

func (m *machine) IP() (net.IP, error) {
	return net.ParseIP(m.server), nil
}

func (m *machine) Type() (vms.MachineType, error) {
	return defaultType, nil
}

func (m *machine) Platform() string {
	return "unknown"
}

func (m *machine) Progress() (uint8, error) {
	return 100, nil
}

func (m *machine) Start() error {
	return nil
}

func (m *machine) Stop() error {
	return nil
}

func (m *machine) Terminate() error {
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

func (m *machine) Id() string {
	return m.id
}

func (m *machine) Name() (string, error) {
	return "Windows Active Directory", nil
}

func (m *machine) Credentials() (string, string, error) {
	return m.user, m.password, nil
}

func (m *machine) Attributes() (vms.MachineAttributes, error) {

	attr := vms.MachineAttributes{}

	Id, err := m.Id()
	if err != nil {
		return nil, err
	}
	Platform, err := m.Platform()
	if err != nil {
		return nil, err
	}
	Name, err := m.Name()
	if err != nil {
		return nil, err
	}
	Status, err := m.Status()
	if err != nil {
		return nil, err
	}
	Ip, err := m.IP()
	if err != nil {
		return nil, err
	}
	Type, err := m.Type()
	if err != nil {
		return nil, err
	}
	Progress, err := m.Progress()
	if err != nil {
		return nil, err
	}
	Username, Password, err := m.Credentials()
	if err != nil {
		return nil, err
	}
	return attr, nil
}
