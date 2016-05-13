package openstack

import "github.com/Nanocloud/community/nanocloud/vms"

type driver struct{}

func (d *driver) Open(options map[string]string) (vms.VM, error) {

	return &vm{
		address:  options["ADDRESS"],
		tenant:   options["TENANT"],
		username: options["USERNAME"],
		password: options["PASSWORD"],
		image:    options["IMAGE"],
	}, nil
}
