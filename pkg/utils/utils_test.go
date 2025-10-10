package utils

import (
	"ibmcloud-vpc-cni-k8s/pkg/metadata"
	"testing"

	"github.com/containernetworking/cni/pkg/skel"
)

type MockMetadata struct{}

func (m *MockMetadata) GetMetadataVNIs(token string) (*metadata.VirtualNetworkInterfaceResponse, error) {
	var vnisResponse = metadata.VirtualNetworkInterfaceResponse{
		VirtualNetworkInterfaces: []struct {
			ID  string `json:"id"`
			IPs []struct {
				Address string `json:"address"`
				ID      string `json:"id"`
			} `json:"ips"`
			Subnet struct {
				CRN string `json:"crn"`
				ID  string `json:"id"`
			}
			PrimaryIP struct {
				Address string `json:"address"`
			} `json:"primary_ip"`
		}{
			{
				ID: "vni-1",
				IPs: []struct {
					Address string `json:"address"`
					ID      string `json:"id"`
				}{
					{Address: "10.0.0.1", ID: "ip-1"},
					{Address: "10.0.0.2", ID: "ip-2"},
				},
				Subnet: struct {
					CRN string `json:"crn"`
					ID  string `json:"id"`
				}{
					CRN: "crn:v1:bluemix:public:vpc::a/12345::subnet:subnet-1",
					ID:  "subnet-1",
				},
				PrimaryIP: struct {
					Address string `json:"address"`
				}{
					Address: "10.0.0.1",
				},
			},
			{
				ID: "vni-2",
				IPs: []struct {
					Address string `json:"address"`
					ID      string `json:"id"`
				}{
					{Address: "10.0.1.2", ID: "ip-3"},
					{Address: "10.0.1.3", ID: "ip-4"},
					{Address: "10.0.1.4", ID: "ip-5"},
				},
				Subnet: struct {
					CRN string `json:"crn"`
					ID  string `json:"id"`
				}{
					CRN: "crn:v1:bluemix:public:vpc::a/12345::subnet:subnet-2",
					ID:  "subnet-2",
				},
				PrimaryIP: struct {
					Address string `json:"address"`
				}{
					Address: "10.0.1.2",
				},
			},
		},
	}
	return &vnisResponse, nil
}

func (m *MockMetadata) GetInstance(auth string) (*metadata.InstanceResponse, error) {
	return &metadata.InstanceResponse{
		Zone: struct {
			Name string `json:"name"`
		}{
			Name: "eu-es-1",
		},
	}, nil
}

func (m *MockMetadata) GetMetadataAccessToken() (string, error) {
	return "mock-access-token", nil
}

func (m *MockMetadata) GetIAMAccessToken(token string) (string, error) {
	return "mock-iam-token", nil
}

func newUtilsWithMock() *Utils {
	return &Utils{md: &MockMetadata{}}
}

func TestGetRegion(t *testing.T) {
	u := newUtilsWithMock()
	region := u.GetRegion()
	expected := "eu-es"
	if region != expected {
		t.Errorf("GetRegion() = %s; want %s", region, expected)
	}
}

func TestGetDefaultSubnetName(t *testing.T) {
	u := newUtilsWithMock()
	subnet := u.GetDefaultSubnetName()
	expected := "ibm-default-gateway"
	if subnet != expected {
		t.Errorf("GetDefaultSubnetName() = %s; want %s", subnet, expected)
	}
}

func TestGetIAMToken(t *testing.T) {
	u := newUtilsWithMock()
	token := u.GetIAMToken()
	expected := "mock-iam-token"
	if token != expected {
		t.Errorf("GetIAMToken() = %s; want %s", token, expected)
	}
}

func TestGetHostVPeer(t *testing.T) {
	containerId := newUtilsWithMock().GetHostVPeer(&skel.CmdArgs{
		ContainerID: "dfer243ds1adf39-324dsf3-dsf3",
	})
	if containerId != "cni0-dfer243ds1" {
		t.Errorf("GetHostVPeer() = %s; want %s", containerId, "cni0-dfer243ds1")
	}
}

func TestGetConfig(t *testing.T) {
	data := []byte(`{
		"cniVersion": "0.4.0",
		"name": "ibm-vpc-cni",
		"type": "ibmcloud-vpc-cni-k8s",
		"ipam": {
		  "type": "static"
		},
		"config": {
		  "network-policy": "ebpf"
		}
	  }`)
	cniConfig := newUtilsWithMock().GetConfig(&skel.CmdArgs{
		StdinData: data,
	})
	if cniConfig.Config.NetworkPolicy != "ebpf" {
		t.Errorf("TestGetConfig() = %s; want %s", cniConfig.Config.NetworkPolicy, "ebpf")
	}

	data = []byte(`{
		"cniVersion": "0.4.0",
		"name": "ibm-vpc-cni",
		"type": "ibmcloud-vpc-cni-k8s",
		"ipam": {
		  "type": "static"
		}
	  }`)
	cniConfig = newUtilsWithMock().GetConfig(&skel.CmdArgs{
		StdinData: data,
	})
	if cniConfig.Config.NetworkPolicy != "" {
		t.Errorf("TestGetConfig() = %s; want %s", cniConfig.Config.NetworkPolicy, "")
	}
	cniConfig = newUtilsWithMock().GetConfig(&skel.CmdArgs{})
	if cniConfig.Config.NetworkPolicy != "" {
		t.Errorf("TestGetConfig() = %s; want %s", cniConfig.Config.NetworkPolicy, "")
	}
}
