package main

import (
	"fmt"
	"ibmcloud-vpc-cni-k8s/pkg/utils"
	"ibmcloud-vpc-cni-k8s/pkg/veth"
	"ibmcloud-vpc-cni-k8s/pkg/vni"
	"ibmcloud-vpc-cni-k8s/pkg/vpc"
	"net"
	"runtime"

	"github.com/containernetworking/cni/pkg/skel"
	types "github.com/containernetworking/cni/pkg/types/040"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var logger = utils.NewLogger()
var vniInstance = vni.NewVNI()
var vpcInstance = vpc.NewVPC()

func cmdAdd(args *skel.CmdArgs) error {
	logger.Log(fmt.Sprintf("New pod: %+v", args))
	vniCandidate := vniInstance.GetVNICandidate()
	reservedID := vpcInstance.ReservedIP(vniCandidate.SubnetID)
	defer func() {
		if r := recover(); r != nil {
			if reservedID != nil {
				vpcInstance.ReleaseIP(vniCandidate.SubnetID, reservedID.ID)
			}
			panic(r)
		}
	}()
	vpcInstance.AddIP(vniCandidate.ID, reservedID.ID)
	ipNet := getIpNet(reservedID.Address, vpcInstance.GetSubnetPrefix(vniCandidate.SubnetID))
	veth.SetupPodVeth(args, ipNet, vpcInstance.GetGatewayIP(vniCandidate.SubnetID), args.Netns)
	result := getResult(ipNet, vniCandidate.SubnetID, vpcInstance)
	logger.Log(fmt.Sprintf("IP: %s has been added to VNI: %s", reservedID.Address, vniCandidate.ID))
	return result.Print()
}

func getResult(ipNet *net.IPNet, subnetID string, vpcInstance vpc.VPC) *types.Result {
	return &types.Result{
		CNIVersion: "0.4.0",
		IPs: []*types.IPConfig{
			{
				Version: "4",
				Address: *ipNet,
				Gateway: vpcInstance.GetGatewayIP(subnetID),
			},
		},
	}
}

func getIpNet(address string, prefix int) *net.IPNet {
	ipNet := &net.IPNet{
		IP:   net.ParseIP(address),
		Mask: net.CIDRMask(prefix, 32),
	}
	return ipNet
}

func cmdDel(args *skel.CmdArgs) error {
	logger.Log(fmt.Sprintf("Del pod: %+v", args))
	origNS, _ := netns.Get() // save current namespace
	podNS, _ := netns.GetFromPath(args.Netns)
	runtime.LockOSThread() // important!
	netns.Set(podNS)
	podLink, _ := netlink.LinkByName(args.IfName)
	addrs, _ := netlink.AddrList(podLink, 2)
	for _, a := range addrs {
		if a.IP.String() != "127.0.0.1" {
			vniIP, subnetID, err := vniInstance.GetIdFromIp(a.IP.String())
			if err == nil && !vniIP.Primary {
				vpcInstance.ReleaseIP(subnetID, vniIP.Id)
				logger.Log(fmt.Sprintf("IP: %s has been deleted on VNI: %s and deleted from subnet: %s", vniIP.Ip, vniIP.Id, subnetID))
			}
		}
	}
	netns.Set(origNS)
	defer podNS.Close()
	defer origNS.Close()
	return nil
}

func main() {
	skel.PluginMainFuncs(skel.CNIFuncs{
		Add: cmdAdd,
		Del: cmdDel,
	}, version.PluginSupports("0.4.0"), "ibmcloud-vpc-cni-k8s")
}
