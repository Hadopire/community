package aws

import (
	crand "crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/Nanocloud/community/nanocloud/vms"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type vm struct {
	sess    *session.Session
	svc     *ec2.EC2
	pemPath string
}

func Decode(encryptedPasswdB64 string, key *rsa.PrivateKey) (string, error) {

	encryptedPasswd, err := base64.StdEncoding.DecodeString(encryptedPasswdB64)
	if err != nil {
		return "", err
	}

	out, err := rsa.DecryptPKCS1v15(nil, key, encryptedPasswd)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func Encode(password string, key *rsa.PublicKey) (string, error) {

	crypted, err := rsa.EncryptPKCS1v15(crand.Reader, key, []byte(password))
	if err != nil {
		return "", err
	}

	out := base64.StdEncoding.EncodeToString(crypted)

	return out, nil
}

func (v *vm) Machines() ([]vms.Machine, error) {

	resp, err := v.svc.DescribeInstances(nil)
	if err != nil {
		return nil, err
	}
	machines := make([]vms.Machine, 0)
	for index := range resp.Reservations {
		for _, instance := range resp.Reservations[index].Instances {
			if *instance.State.Code != 48 {
				machines = append(machines, &machine{
					id:      *instance.InstanceId,
					svc:     v.svc,
					pemPath: v.pemPath,
				})
			}
		}
	}
	return machines, nil
}

func (v *vm) Machine(id string) (vms.Machine, error) {

	resp, err := v.svc.DescribeInstances(nil)
	if err != nil {
		return nil, err
	}

	for index := range resp.Reservations {
		for _, instance := range resp.Reservations[index].Instances {
			if *instance.InstanceId == id {
				return &machine{
					id:      id,
					svc:     v.svc,
					pemPath: v.pemPath,
				}, nil
			}
		}
	}
	return nil, errors.New("Instance not found")
}

func (v *vm) Create(attr vms.MachineAttributes) (vms.Machine, error) {

	if attr.Type == nil {
		attr.Type = defaultType
	}

	t, ok := attr.Type.(*machineType)
	if !ok {
		return nil, errors.New("VM Type not supported")
	}

	if _, err := os.Stat(v.pemPath); os.IsNotExist(err) {

		cResp, err := v.svc.CreateKeyPair(&ec2.CreateKeyPairInput{
			KeyName: aws.String("NanocloudKey"),
		})
		if err != nil {

			_, err = v.svc.DeleteKeyPair(&ec2.DeleteKeyPairInput{
				KeyName: aws.String("NanocloudKey"),
			})
			if err != nil {
				return nil, err
			}

			cResp, err = v.svc.CreateKeyPair(&ec2.CreateKeyPairInput{
				KeyName: aws.String("NanocloudKey"),
			})
			if err != nil {
				return nil, err
			}
		}

		fo, err := os.Create(v.pemPath)
		if err != nil {
			return nil, err
		}
		if _, err = fo.Write([]byte(*cResp.KeyMaterial)); err != nil {
			return nil, err
		}

		fo.Close()
	}

	params := &ec2.RunInstancesInput{
		ImageId:               aws.String(t.ami),
		MaxCount:              aws.Int64(1),
		MinCount:              aws.Int64(1),
		DisableApiTermination: aws.Bool(false),
		KeyName:               aws.String("NanocloudKey"),
		InstanceType:          aws.String(t.instanceType),
		UserData:              aws.String(base64.StdEncoding.EncodeToString([]byte("<powershell>Invoke-WebRequest https://s3-eu-west-1.amazonaws.com/nanocloud/plaza.exe -OutFile C:\\plaza.exe\nC:\\plaza.exe\nrm C:\\plaza.exe\nNew-NetFirewallRule -Protocol TCP -LocalPort 9090 -Direction Inbound -Action Allow -DisplayName PLAZA</powershell>"))),
	}
	resp, err := v.svc.RunInstances(params)

	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	tagParams := &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(*resp.Instances[0].InstanceId),
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(attr.Name),
			},
		},
	}
	_, err = v.svc.CreateTags(tagParams)

	if err != nil {
		return nil, err
	}
	//mac, _ := v.Machine(*resp.Instances[0].InstanceId)
	//p := provisioner.New(plaza.Provision(mac))
	//go p.Run()
	//p.AddOutput(os.Stdout)
	return v.Machine(*resp.Instances[0].InstanceId)
}

func (v *vm) Types() ([]vms.MachineType, error) {
	types := make([]vms.MachineType, 0)
	for _, it := range instanceTypes {
		types = append(types, &machineType{
			ami:          "ami-3acf2f55", // windows 2012 r2
			instanceType: it,
		})
	}
	return types, nil
}

func (v *vm) Type(id string) (vms.MachineType, error) {
	for _, it := range instanceTypes {
		if it == id {
			return &machineType{
				ami:          "ami-3acf2f55",
				instanceType: it,
			}, nil
		}
	}
	return nil, errors.New("Type does not exists")
}
