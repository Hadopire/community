package openstack

type machineType struct {
	flavor     string
	flavorHref string
	image      string
}

func (t *machineType) GetID() string {
	return t.flavor
}
