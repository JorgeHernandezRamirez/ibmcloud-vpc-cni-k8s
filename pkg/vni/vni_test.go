package vni

import (
	"ibmcloud-vpc-cni-k8s/pkg/metadata"
	"testing"
)

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

type MockMetadata struct{}

func (m *MockMetadata) GetIAMAccessToken(auth string) (string, error) {
	return "iamAccessToken", nil
}

func (m *MockMetadata) GetInstance(auth string) (*metadata.InstanceResponse, error) {
	return &metadata.InstanceResponse{
		Zone: struct {
			Name string `json:"name"`
		}{
			Name: "eu-es",
		},
	}, nil
}

func (m *MockMetadata) GetMetadataAccessToken() (string, error) {
	return "metadataAccessToken", nil
}

func (m *MockMetadata) GetMetadataVNIs(token string) (*metadata.VirtualNetworkInterfaceResponse, error) {
	return &vnisResponse, nil
}

func newVNIWithMock() *VNI {
	v := &VNI{}
	v.md = &MockMetadata{}
	return v
}

func TestConvertVNIs(t *testing.T) {
	vnis := *ConvertVNIs(&vnisResponse)
	if len(vnis) != 2 {
		t.Fatalf("expected 2 VNI, got %d", len(vnis))
	}
	if vnis[0].ID != "vni-1" || vnis[0].SubnetID != "subnet-1" {
		t.Errorf("unexpected VNI data %+v", vnis[0])
	}
	if !vnis[0].IPs[0].Primary || vnis[0].IPs[1].Primary || len(vnis[0].IPs) != 2 {
		t.Errorf("primary IP flags incorrect %+v", vnis[0].IPs)
	}
	if vnis[1].ID != "vni-2" || vnis[1].SubnetID != "subnet-2" {
		t.Errorf("unexpected VNI data %+v", vnis[0])
	}
	if !vnis[1].IPs[0].Primary || vnis[1].IPs[1].Primary || len(vnis[1].IPs) != 3 {
		t.Errorf("primary IP flags incorrect %+v", vnis[0].IPs)
	}
}

func TestGetVNIs(t *testing.T) {
	v := newVNIWithMock()
	vnis := *v.GetVNIs()
	if len(vnis) != 2 {
		t.Fatalf("expected 2 VNIs, got %d", len(vnis))
	}
}

func TestGetIdFromIp(t *testing.T) {
	v := newVNIWithMock()
	ipInfo, subnetID, err := v.GetIdFromIp("10.0.0.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ipInfo.Id != "ip-2" || subnetID != "subnet-1" {
		t.Errorf("unexpected result: %+v %s", ipInfo, subnetID)
	}

	_, _, err = v.GetIdFromIp("10.0.9.9")
	if err == nil {
		t.Errorf("expected error for unknown IP")
	}
}

func TestGetVNICandidate(t *testing.T) {
	v := newVNIWithMock()
	candidate := v.GetVNICandidate()
	if candidate.ID != "vni-1" {
		t.Errorf("expected vni-2 as candidate (least IPs), got %s", candidate.ID)
	}
}
