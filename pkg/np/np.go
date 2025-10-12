package np

import (
	"fmt"
	"ibmcloud-vpc-cni-k8s/pkg/utils"
	"os/exec"

	"github.com/containernetworking/cni/pkg/skel"
)

type NetworkPolicyInterface interface {
	Configure(args *skel.CmdArgs, ip string)
	Deconfigure(args *skel.CmdArgs)
}

type NoneNetworkPolicy struct {
}

func (p *NoneNetworkPolicy) Configure(args *skel.CmdArgs, ip string) {
}

func (p *NoneNetworkPolicy) Deconfigure(args *skel.CmdArgs) {
}

type EBPFNetworkPolicy struct {
	utils  utils.UtilsInterface
	logger *utils.Logger
}

func NewNetworkPolicy(args *skel.CmdArgs) NetworkPolicyInterface {
	logger := utils.NewLogger()
	u := utils.NewUtils()
	if u.GetConfig(args).Config.NetworkPolicy == "ebpf" {
		logger.Log(fmt.Sprintf("EBPF loading..."))
		return &EBPFNetworkPolicy{
			utils:  u,
			logger: logger,
		}
	}
	logger.Log(fmt.Sprintf("No EBPF"))
	return &NoneNetworkPolicy{}
}

func (p *EBPFNetworkPolicy) Configure(args *skel.CmdArgs, ip string) {
	p.logger.Log(fmt.Sprintf("Configuring IP %s on dev %s", p.utils.GetHostVPeer(args), ip))
	cmd := exec.Command("/opt/ebpf/config", "create", p.utils.GetHostVPeer(args), ip)
	_, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("Error: %v", err))
	}
	p.logger.Log(fmt.Sprintf("Done"))
}

func (p *EBPFNetworkPolicy) Deconfigure(args *skel.CmdArgs) {
	p.logger.Log(fmt.Sprintf("Deconfiguring on dev %s", p.utils.GetHostVPeer(args)))
	cmd := exec.Command("/opt/ebpf/config", "delete", p.utils.GetHostVPeer(args))
	_, err := cmd.CombinedOutput()
	if err != nil {
		p.logger.Log(fmt.Sprintf("Error: %v", err))
	}
	p.logger.Log(fmt.Sprintf("Done"))
}
