package vni

import (
	"fmt"
	"ibmcloud-vpc-cni-k8s/pkg/metadata"
)

type VirtualNetworkInterfaceIP struct {
	Id      string
	Ip      string
	Primary bool
}

type VirtualNetworkInterface struct {
	ID       string
	SubnetID string
	IPs      []VirtualNetworkInterfaceIP
}

type VNI struct {
	md metadata.MetadataInterface
}

func NewVNI() *VNI {
	return &VNI{md: metadata.NewMetadataService()}
}

func (vni *VNI) GetVNIs() *[]VirtualNetworkInterface {
	metadataAccessToken, _ := vni.md.GetMetadataAccessToken()
	metadataVNIs, _ := vni.md.GetMetadataVNIs(metadataAccessToken)
	return ConvertVNIs(metadataVNIs)
}

func (vni *VNI) GetIdFromIp(ip string) (*VirtualNetworkInterfaceIP, string, error) {
	for _, v := range *vni.GetVNIs() {
		for _, i := range v.IPs {
			if i.Ip == ip {
				return &i, v.SubnetID, nil
			}
		}
	}
	return &VirtualNetworkInterfaceIP{}, "", fmt.Errorf("no ID found for IP %s", ip)
}

/*
*
This function returns a VNI to which a new IP will be bound.
The current algorithm selects the VNI with the fewest IPs
*/
func (vni *VNI) GetVNICandidate() *VirtualNetworkInterface {
	vnis := *vni.GetVNIs()
	vniToReturn := vnis[0]
	for _, v := range vnis {
		if len(v.IPs) < len(vniToReturn.IPs) {
			vniToReturn = v
		}
	}
	return &vniToReturn
}

func ConvertVNIs(metaResp *metadata.VirtualNetworkInterfaceResponse) *[]VirtualNetworkInterface {
	var result []VirtualNetworkInterface

	for _, vni := range metaResp.VirtualNetworkInterfaces {
		var ips []VirtualNetworkInterfaceIP
		for _, ip := range vni.IPs {
			ips = append(ips, VirtualNetworkInterfaceIP{Id: ip.ID, Ip: ip.Address, Primary: vni.PrimaryIP.Address == ip.Address})
		}

		result = append(result, VirtualNetworkInterface{
			ID:       vni.ID,
			SubnetID: vni.Subnet.ID,
			IPs:      ips,
		})
	}
	return &result
}
