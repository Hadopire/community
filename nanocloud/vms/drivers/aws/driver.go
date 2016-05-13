package aws

import (
	"github.com/Nanocloud/community/nanocloud/vms"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type driver struct{}

func (d *driver) Open(options map[string]string) (vms.VM, error) {

	config := aws.NewConfig().
		WithRegion(options["REGION"]).
		WithCredentials(
			credentials.NewStaticCredentials(
				options["ACCESS"],
				options["SECRET"],
				"",
			),
		)
	s := session.New(config)
	defaultType.ami = options["AMI"]
	defaultType.instanceType = options["INSTANCE_TYPE"]
	return &vm{
		sess:    s.Copy(),
		svc:     ec2.New(s),
		pemPath: options["PEM_PATH"],
	}, nil
}
