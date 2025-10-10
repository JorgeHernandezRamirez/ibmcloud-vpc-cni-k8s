package pod

import (
	"ibmcloud-vpc-cni-k8s/pkg/utils"
	"ibmcloud-vpc-cni-k8s/pkg/vni"
	"ibmcloud-vpc-cni-k8s/pkg/vpc"
	"net"
	"testing"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/stretchr/testify/mock"
)

type MockVNI struct {
}

func (v *MockVNI) GetVNIs() *[]vni.VirtualNetworkInterface {
	return &[]vni.VirtualNetworkInterface{
		{
			ID:       "vni01",
			SubnetID: "subnet01",
			IPs: []vni.VirtualNetworkInterfaceIP{
				{
					Id:      "ip01",
					Ip:      "10.1.1.4",
					Primary: true,
				},
				{
					Id:      "ip02",
					Ip:      "10.1.1.5",
					Primary: false,
				},
			},
		},
		{
			ID:       "vni02",
			SubnetID: "subnet02",
			IPs: []vni.VirtualNetworkInterfaceIP{
				{
					Id:      "ip03",
					Ip:      "10.1.1.6",
					Primary: true,
				},
			},
		},
	}
}

func (v *MockVNI) GetIdFromIp(ip string) (*vni.VirtualNetworkInterfaceIP, string, error) {
	return &vni.VirtualNetworkInterfaceIP{
		Id:      "ip01",
		Ip:      ip,
		Primary: false,
	}, "", nil
}

func (v *MockVNI) GetVNICandidate() *vni.VirtualNetworkInterface {
	a := *v.GetVNIs()
	return &a[0]
}

type MockVeth struct {
}

func (v *MockVeth) SetupPodVeth(args *skel.CmdArgs, ipNet *net.IPNet, ipNetGateway net.IP, netnsPath string) {
}

type MockVPC struct {
	mock.Mock
}

func (m *MockVPC) ReservedIP(subnetID string) *vpc.VPCReservedIP {
	args := m.Called(subnetID)
	return args.Get(0).(*vpc.VPCReservedIP)
}
func (m *MockVPC) ReleaseIP(subnetID string, reservedIP string) {}
func (m *MockVPC) AddIP(vniID string, reservedIP string) {
	m.Called(vniID, reservedIP)
}
func (m *MockVPC) GetGatewayIP(subnetID string) net.IP            { return net.ParseIP("10.1.0.1") }
func (m *MockVPC) GetSubnetPrefix(subnetID string) int            { return 24 }
func (m *MockVPC) AutoRelease(subnetID string, reservedID string) {}

func TestPodCreate(t *testing.T) {
	mockVPC := prepareVPCMock()
	pod := &Pod{
		logger:   utils.NewLogger(),
		vethInst: &MockVeth{},
		vpcInst:  mockVPC,
		vniInst:  &MockVNI{},
	}
	result := pod.Create(&skel.CmdArgs{ContainerID: "cont_id_01", StdinData: []byte("{}")})
	if result.IPs[0].Address.String() != "10.1.0.6/24" {
		t.Errorf("primary IP flags incorrect %+v", result.IPs[0].Address.String())
	}
	if result.IPs[0].Gateway.String() != "10.1.0.1" {
		t.Errorf("primary IP flags incorrect %+v", result.IPs[0].Address.String())
	}
	mockVPC.AssertCalled(t, "AddIP", "vni01", "ip04")
}

func TestPodDelete(t *testing.T) {
	mockVPC := prepareVPCMock()
	pod := &Pod{
		logger:   utils.NewLogger(),
		vethInst: &MockVeth{},
		vpcInst:  mockVPC,
		vniInst:  &MockVNI{},
	}
	pod.Delete(&skel.CmdArgs{ContainerID: "cont_id_01"})
}

func prepareVPCMock() *MockVPC {
	mock := &MockVPC{}
	mock.On("AddIP", "vni01", "ip04").Return()
	mock.On("ReservedIP", "subnet01").Return(&vpc.VPCReservedIP{
		ID:      "ip04",
		Address: "10.1.0.6",
	})
	return mock
}
