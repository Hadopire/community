package machinebroker

import (
	"time"

	"github.com/Nanocloud/community/nanocloud/connectors/db"
	"github.com/Nanocloud/community/nanocloud/connectors/vms"
	"github.com/Nanocloud/community/nanocloud/models/users"
	vm "github.com/Nanocloud/community/nanocloud/vms"
)

func FindAvailableMachine() (string, error) {

	rows, err := db.Query(`
		SELECT machine_id FROM machines_users
		WHERE user_id isnull
	`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var machineId string
		rows.Scan(
			&machineId,
		)

		return machineId, nil
	}

	err = rows.Err()
	if err != nil {
		return "", err
	}
	return "", nil
}

func FindUserMachine(userId string) (string, error) {

	rows, err := db.Query(`
		SELECT machine_id FROM machines_users
		WHERE user_id=$1::varchar
	`, userId)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var machineId string
		rows.Scan(
			&machineId,
		)

		return machineId, nil
	}

	err = rows.Err()
	if err != nil {
		return "", err
	}
	return "", nil
}

func GetMachine(user *users.User) (vm.Machine, error) {

	var machine vm.Machine
	machineId, err := FindUserMachine(user.GetID())
	if err != nil {
		return nil, err
	}

	if machineId != "" {

		machine, err = vms.Machine(machineId)
		if err != nil {
			return nil, err
		}
	} else {

		machineId, err = FindAvailableMachine()
		if err != nil {
			return nil, err
		}

		if machineId != "" {
			_, err = db.Exec("UPDATE machines_users SET user_id=$1::varchar WHERE machine_id=$2::varchar",
				user.GetID(), machineId)
			if err != nil {
				return nil, err
			}

			err = UpgradePool(1)
			if err != nil {
				return nil, err
			}

			machine, err = vms.Machine(machineId)
			if err != nil {
				return nil, err
			}
		}
	}

	if machine == nil {
		err = UpgradePool(3)
		if err != nil {
			return nil, err
		}
		return GetMachine(user)
	}

	status, err := machine.Status()
	if err != nil {
		return nil, err
	}

	for status != vm.StatusUp {
		if status == vm.StatusDown {
			machine.Start()
		}
		time.Sleep(time.Millisecond * 500)
		status, err = machine.Status()
		if err != nil {
			return nil, err
		}
	}
	return machine, nil
}

func UpgradePool(nb uint) error {
	for i := uint(0); i < nb; i++ {

		attr := vm.MachineAttributes{
			Type:     nil,
			Name:     "",
			Username: "",
			Password: "",
			Ip:       "",
		}
		m, err := vms.Create(attr)
		if err != nil {
			return err
		}
		_, err = db.Query(
			`INSERT INTO machines_users
				(machine_id)
				VALUES ( $1::varchar )
				`, m.Id(),
		)
	}
	return nil
}

func PoolEmpty() (bool, error) {

	rows, err := db.Query("SELECT user_id FROM machines_users")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var userId string
		rows.Scan(
			&userId,
		)
		if userId == "" {
			return false, nil
		}
	}
	return true, nil
}
