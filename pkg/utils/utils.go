package utils

import (
	"encoding/json"
	"fmt"
	"ibmcloud-vpc-cni-k8s/pkg/metadata"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
)

type Config struct {
	NetworkPolicy string `json:"network-policy"`
}

type CNIConfig struct {
	Config Config `json:"config"`
}

type UtilsInterface interface {
	GetRegion() string
	GetDefaultSubnetName() string
	GetIAMToken() string
	GetHostVPeer(args *skel.CmdArgs) string
	GetConfig(args *skel.CmdArgs) *CNIConfig
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

func (v *Utils) GetHostVPeer(args *skel.CmdArgs) string {
	return fmt.Sprintf("cni0-%s", args.ContainerID[0:10])
}

func (v *Utils) GetConfig(args *skel.CmdArgs) *CNIConfig {
	var cni CNIConfig
	if args.StdinData == nil {
		return &CNIConfig{}
	}
	err := json.Unmarshal(args.StdinData, &cni)
	if err != nil {
		panic(err)
	}
	return &cni
}
