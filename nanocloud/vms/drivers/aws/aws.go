package aws

import "github.com/Nanocloud/community/nanocloud/vms"

func init() {
	vms.Register("aws", &driver{})
}
