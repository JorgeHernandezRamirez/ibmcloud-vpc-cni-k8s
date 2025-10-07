package vpc

import (
	"fmt"
	"ibmcloud-vpc-cni-k8s/pkg/utils"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
)

type VPCInterface interface {
	ReservedIP(subnetID string) *VPCReservedIP
	ReleaseIP(subnetID string, reservedIP string)
	AddIP(vniID string, reservedIP string)
	AutoRelease(subnetID string, reservedID string)
	GetGatewayIP(subnetID string) net.IP
	GetSubnetPrefix(subnetID string) int
}

type VPCReservedIP struct {
	ID      string
	Address string
}
type SdkVPC struct {
	vpcV1 *vpcv1.VpcV1
	utils utils.UtilsInterface
}

func NewVPC() VPCInterface {
	utilsInstance := utils.NewUtils()
	authenticator := &core.BearerTokenAuthenticator{
		BearerToken: utilsInstance.GetIAMToken(),
	}
	serviceURL, _ := vpcv1.GetServiceURLForRegion(utilsInstance.GetRegion())
	options := &vpcv1.VpcV1Options{
		Authenticator: authenticator,
		URL:           serviceURL,
	}
	vpcV1, _ := vpcv1.NewVpcV1(options)
	return &SdkVPC{
		vpcV1: vpcV1,
		utils: utils.NewUtils(),
	}
}

func (v *SdkVPC) ReservedIP(subnetID string) *VPCReservedIP {
	var reservedIP = v.doReservedIP(subnetID)
	for {
		reservedIP = v.getReservedIP(subnetID, *reservedIP.ID)
		if *reservedIP.LifecycleState == "available" || *reservedIP.LifecycleState == "stable" {
			break
		}
		time.Sleep(5 * time.Second)
	}
	return &VPCReservedIP{
		ID:      *reservedIP.ID,
		Address: *reservedIP.Address,
	}
}

func (v *SdkVPC) doReservedIP(subnetID string) *vpcv1.ReservedIP {
	reservedIP, _, err := v.vpcV1.CreateSubnetReservedIP(&vpcv1.CreateSubnetReservedIPOptions{
		SubnetID: &subnetID,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create reserved IP: %v", err))
	}
	return reservedIP
}

func (v *SdkVPC) getReservedIP(subnetID string, reservedIPID string) *vpcv1.ReservedIP {
	reservedIPResponse, _, err := v.vpcV1.GetSubnetReservedIP(&vpcv1.GetSubnetReservedIPOptions{
		SubnetID: &subnetID,
		ID:       &reservedIPID,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to get reserved IP: %v", err))
	}
	return reservedIPResponse
}

func (v *SdkVPC) AddIP(vniID string, reservedIP string) {
	_, _, err := v.vpcV1.AddVirtualNetworkInterfaceIP(&vpcv1.AddVirtualNetworkInterfaceIPOptions{
		VirtualNetworkInterfaceID: &vniID,
		ID:                        &reservedIP,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to add IP: %v", err))
	}
}

func (v *SdkVPC) AutoRelease(subnetID string, reservedID string) {
	_, _, err := v.vpcV1.UpdateSubnetReservedIP(&vpcv1.UpdateSubnetReservedIPOptions{
		SubnetID: &subnetID,
		ID:       &reservedID,
		ReservedIPPatch: map[string]interface{}{
			"auto_delete": true,
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to autorelease IP: %v", err))
	}
}

func (v *SdkVPC) ReleaseIP(subnetID string, reservedIP string) {
	_, err := v.vpcV1.DeleteSubnetReservedIP(&vpcv1.DeleteSubnetReservedIPOptions{
		SubnetID: &subnetID,
		ID:       &reservedIP,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to create reserved IP: %v", err))
	}
}

func (v *SdkVPC) GetGatewayIP(subnetID string) net.IP {
	name := v.utils.GetDefaultSubnetName()
	reservedIps, _, _ := v.vpcV1.ListSubnetReservedIps(&vpcv1.ListSubnetReservedIpsOptions{
		SubnetID: &subnetID,
	})
	for _, ip := range reservedIps.ReservedIps {
		if *ip.Name == name {
			return net.ParseIP(*ip.Address)
		}
	}
	panic(fmt.Sprintf("no reserved IP found with name %s", name))
}

func (v *SdkVPC) GetSubnetPrefix(subnetID string) int {
	response, _, _ := v.vpcV1.GetSubnet(&vpcv1.GetSubnetOptions{
		ID: &subnetID,
	})
	parts := strings.Split(*response.Ipv4CIDRBlock, "/")
	prefix, _ := strconv.Atoi(parts[1])
	return prefix
}
