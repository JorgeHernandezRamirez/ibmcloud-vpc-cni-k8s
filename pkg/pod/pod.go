package pod

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
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type Pod struct {
	logger   *utils.Logger
	vniInst  *vni.VNI
	vethInst *veth.Veth
	vpcInst  vpc.VPC
}

func NewPod() *Pod {
	return &Pod{
		logger:   utils.NewLogger(),
		vniInst:  vni.NewVNI(),
		vethInst: veth.NewVeth(),
		vpcInst:  vpc.NewVPC(),
	}
}

func (p *Pod) Create(args *skel.CmdArgs) *types.Result {
	p.logger.Log(fmt.Sprintf("New pod: %+v", args))
	vniCandidate := p.vniInst.GetVNICandidate()
	reservedID := p.vpcInst.ReservedIP(vniCandidate.SubnetID)
	defer func() {
		if r := recover(); r != nil {
			if reservedID != nil {
				p.vpcInst.ReleaseIP(vniCandidate.SubnetID, reservedID.ID)
			}
			panic(r)
		}
	}()
	p.vpcInst.AddIP(vniCandidate.ID, reservedID.ID)
	ipNet := getIpNet(reservedID.Address, p.vpcInst.GetSubnetPrefix(vniCandidate.SubnetID))
	p.vethInst.SetupPodVeth(args, ipNet, p.vpcInst.GetGatewayIP(vniCandidate.SubnetID), args.Netns)
	result := getResult(ipNet, vniCandidate.SubnetID, p.vpcInst)
	p.logger.Log(fmt.Sprintf("IP: %s has been added to VNI: %s", reservedID.Address, vniCandidate.ID))
	return result

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

func (p *Pod) Delete(args *skel.CmdArgs) {
	p.logger.Log(fmt.Sprintf("Del pod: %+v", args))
	origNS, _ := netns.Get() // save current namespace
	podNS, _ := netns.GetFromPath(args.Netns)
	defer podNS.Close()
	defer origNS.Close()
	runtime.LockOSThread() // important!
	netns.Set(podNS)
	podLink, _ := netlink.LinkByName(args.IfName)
	addrs, _ := netlink.AddrList(podLink, 2)
	p.releaseIPFromAddrs(addrs)
	netns.Set(origNS)
}

func (p *Pod) releaseIPFromAddrs(addrs []netlink.Addr) {
	for _, a := range addrs {
		if a.IP.String() != "127.0.0.1" {
			vniIP, subnetID, err := p.vniInst.GetIdFromIp(a.IP.String())
			if err == nil && !vniIP.Primary {
				p.vpcInst.ReleaseIP(subnetID, vniIP.Id)
				p.logger.Log(fmt.Sprintf("IP: %s has been deleted on VNI: %s and deleted from subnet: %s", vniIP.Ip, vniIP.Id, subnetID))
			}
		}
	}
}
