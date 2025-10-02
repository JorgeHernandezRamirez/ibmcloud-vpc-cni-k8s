package utils

import (
	"ibmcloud-vpc-cni-k8s/pkg/metadata"
	"strings"
)

func GetRegion() string {
	metadata := metadata.NewMetadataService()
	accessToken, _ := metadata.GetMetadataAccessToken()
	instance, _ := metadata.GetInstance(accessToken)
	parts := strings.Split(instance.Zone.Name, "-")
	if len(parts) >= 2 {
		return strings.Join(parts[:len(parts)-1], "-")
	}
	panic("Cannot determine region")
}

func GetDefaultSubnetName() string {
	return "ibm-default-gateway"
}

func GetIAMToken() string {
	metadata := metadata.NewMetadataService()
	accessToken, _ := metadata.GetMetadataAccessToken()
	iamToken, _ := metadata.GetIAMAccessToken(accessToken)
	return iamToken
}
