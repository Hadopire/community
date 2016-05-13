package aws

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"net"

	"github.com/Nanocloud/community/nanocloud/vms"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type machine struct {
	id       string
	svc      *ec2.EC2
	platform string
	name     string
	ip       net.IP
	mType    vms.MachineType
	pemPath  string
}

func (m *machine) Id() string {
	return m.id
}

func (m *machine) Platform() string {
	return m.platform
}

func (m *machine) Name() (string, error) {

	resp, err := m.svc.DescribeInstances(nil)
	if err != nil {
		return "", err
	}

	for index := range resp.Reservations {
		for _, instance := range resp.Reservations[index].Instances {
			if *instance.InstanceId == m.id {
				for _, keys := range instance.Tags {
					if *keys.Key == "Name" {
						return *keys.Value, nil
					}
				}
			}
		}
	}
	return "", nil
}

func (m *machine) Status() (vms.MachineStatus, error) {

	resp, err := m.svc.DescribeInstances(nil)
	if err != nil {
		return vms.StatusUnknown, err
	}

	statusResp, err := m.svc.DescribeInstanceStatus(&ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{
			aws.String(m.id),
		},
	})
	if err != nil {
		return vms.StatusUnknown, err
	}

	for index := range resp.Reservations {
		for _, instance := range resp.Reservations[index].Instances {
			if *instance.InstanceId == m.id {
				m.platform = *instance.Platform
				if instance.PublicIpAddress != nil && len(*instance.PublicIpAddress) > 0 {
					m.ip = net.ParseIP(*instance.PublicIpAddress)
				} else {
					m.ip = nil
				}
				m.mType = &machineType{
					ami:          *instance.ImageId,
					instanceType: *instance.InstanceType,
				}
				switch *instance.State.Code {
				case 0:
					return vms.StatusBooting, nil
				case 16:
					if _, _, err = m.Credentials(); err != nil {
						return vms.StatusBooting, nil
					} else if *statusResp.InstanceStatuses[0].InstanceStatus.Status != ec2.SummaryStatusOk {
						return vms.StatusBooting, nil
					}
					return vms.StatusUp, nil
				case 32:
					return vms.StatusDown, nil
				case 48:
					return vms.StatusTerminated, nil
				case 64:
					return vms.StatusStopping, nil
				case 80:
					return vms.StatusDown, nil
				default:
					return vms.StatusUnknown, nil
				}
			}
		}
	}
	return vms.StatusUnknown, errors.New("Instance not found")
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

func (m *machine) Start() error {
	_, err := m.svc.StartInstances(&ec2.StartInstancesInput{
		InstanceIds: []*string{
			aws.String(m.id),
		},
	})

	if err != nil {
		return err
	}
	return nil
}

func (m *machine) Stop() error {
	_, err := m.svc.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: []*string{
			aws.String(m.id),
		},
	})

	if err != nil {
		return err
	}
	return nil
}

func (m *machine) Terminate() error {
	_, err := m.svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(m.id),
		},
	})

	if err != nil {
		return err
	}
	return nil
}

func (m *machine) Credentials() (string, string, error) {

	b, err := ioutil.ReadFile(m.pemPath)
	if err != nil {
		return "", "", err
	}

	pResp, err := m.svc.GetPasswordData(&ec2.GetPasswordDataInput{
		InstanceId: aws.String(m.id),
	})

	if err != nil {
		return "", "", err
	}

	cryptedPassword := *pResp.PasswordData

	if cryptedPassword == "" {
		return "", "", errors.New("Password not yet generated")
	}

	block, _ := pem.Decode([]byte(b))
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", "", err
	}

	clearPassword, err := Decode(cryptedPassword, key)
	return "Administrator", clearPassword, nil
}
