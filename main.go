package main

import (
	"ibmcloud-vpc-cni-k8s/pkg/pod"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
)

var podInst = pod.NewPod()

func cmdAdd(args *skel.CmdArgs) error {
	result := podInst.Create(args)
	return result.Print()
}

func cmdDel(args *skel.CmdArgs) error {
	podInst.Delete(args)
	return nil
}

func main() {
	skel.PluginMainFuncs(skel.CNIFuncs{
		Add: cmdAdd,
		Del: cmdDel,
	}, version.PluginSupports("0.4.0"), "ibmcloud-vpc-cni-k8s")
}
