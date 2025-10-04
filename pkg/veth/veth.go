package veth

import (
	"fmt"
	"net"
	"os"
	"runtime"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type VethInterface interface {
	SetupPodVeth(args *skel.CmdArgs, ipNet *net.IPNet, ipNetGateway net.IP, netnsPath string)
}

type Veth struct {
}

func NewVeth() VethInterface {
	return &Veth{}
}

func (v *Veth) SetupPodVeth(args *skel.CmdArgs, ipNet *net.IPNet, ipNetGateway net.IP, netnsPath string) {
	hostVeth := v.createHostInterfaces(args)
	v.configureHostProxyArp(hostVeth)
	v.configureHostRoute(hostVeth, *ipNet)
	origNS, _ := netns.Get() //Save current namespace
	v.configurePodInterface(args, ipNet, ipNetGateway, netnsPath)
	defer netns.Set(origNS) //Back to previous namespace
}

func (v *Veth) configurePodInterface(args *skel.CmdArgs, ipNet *net.IPNet, ipNetGateway net.IP, netnsPath string) {
	podNS, err := netns.GetFromPath(netnsPath) // 2. Get pod network namespace
	if err != nil {
		panic(err)
	}
	defer podNS.Close()
	podLink, err := netlink.LinkByName(args.IfName) // 3. Get pod side link
	if err != nil {
		panic(err)
	}
	if err := netlink.LinkSetNsFd(podLink, int(podNS)); err != nil { // 4. Move pod side into pod netns
		panic(err)
	}
	// 5. Configure pod side inside its netns
	runtime.LockOSThread()     // important!
	origNS, err := netns.Get() // save current namespace
	if err != nil {
		panic(err)
	}
	defer origNS.Close()

	if err := netns.Set(podNS); err != nil {
		panic(err)
	}
	// Bring pod veth up and assign IP
	podLink, err = netlink.LinkByName(args.IfName)
	if err != nil {
		panic(err)
	}
	netlink.LinkSetUp(podLink)
	netlink.AddrAdd(podLink, &netlink.Addr{IPNet: ipNet})
	route := &netlink.Route{
		LinkIndex: podLink.Attrs().Index,
		Gw:        ipNetGateway,
	}
	if err := netlink.RouteAdd(route); err != nil {
		panic(fmt.Sprintf("failed to add default route: %v", err))
	}
}

func (v *Veth) createHostInterfaces(args *skel.CmdArgs) netlink.Link {
	hostVethName := v.getContainerId(args)
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: args.IfName,
		},
		PeerName: hostVethName,
	}
	if err := netlink.LinkAdd(veth); err != nil {
		panic(err)
	}
	netlink.LinkSetUp(veth)
	hostVeth, _ := netlink.LinkByName(hostVethName)
	netlink.LinkSetUp(hostVeth)
	return hostVeth
}

func (v *Veth) configureHostProxyArp(hostVeth netlink.Link) {
	path := fmt.Sprintf("/proc/sys/net/ipv4/conf/%s/proxy_arp", hostVeth.Attrs().Name)
	err := os.WriteFile(path, []byte("1"), 0644)
	if err != nil {
		panic(fmt.Sprintf("failed to enable proxy_arp on %s: %v", hostVeth.Attrs().Name, err))
	}
}

func (v *Veth) configureHostRoute(hostVeth netlink.Link, ipNet net.IPNet) {
	route1 := &netlink.Route{
		LinkIndex: hostVeth.Attrs().Index,
		Dst: &net.IPNet{
			IP:   ipNet.IP,
			Mask: net.CIDRMask(32, 32),
		},
	}
	if err := netlink.RouteAdd(route1); err != nil {
		panic(fmt.Sprintf("failed to add route: %v", err))
	}
}

func (v *Veth) getContainerId(args *skel.CmdArgs) string {
	return fmt.Sprintf("cni0-%s", args.ContainerID[0:10])
}
