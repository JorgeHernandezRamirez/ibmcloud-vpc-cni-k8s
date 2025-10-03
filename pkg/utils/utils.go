package utils

import (
	"ibmcloud-vpc-cni-k8s/pkg/metadata"
	"strings"
)

type UtilsInterface interface {
	GetRegion() string
	GetDefaultSubnetName() string
	GetIAMToken() string
}

type Utils struct {
	md metadata.MetadataInterface
}

func NewUtils() UtilsInterface {
	return &Utils{md: metadata.NewMetadataService()}
}

func (u *Utils) GetRegion() string {
	accessToken, _ := u.md.GetMetadataAccessToken()
	instance, _ := u.md.GetInstance(accessToken)
	parts := strings.Split(instance.Zone.Name, "-")
	if len(parts) >= 2 {
		return strings.Join(parts[:len(parts)-1], "-")
	}
	panic("Cannot determine region")
}

func (u *Utils) GetDefaultSubnetName() string {
	return "ibm-default-gateway"
}

func (u *Utils) GetIAMToken() string {
	accessToken, _ := u.md.GetMetadataAccessToken()
	iamToken, _ := u.md.GetIAMAccessToken(accessToken)
	return iamToken
}
