package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"log"
	"net"
	"strings"
)

type Response struct {
	Message string `json:"message"`
}

func NewEC2Session(region string) (context.Context, *ec2.Client, error) {
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, nil, err
	}
	svc := ec2.NewFromConfig(cfg)
	return ctx, svc, nil
}

func GetVPNEndpointID(ctx context.Context, svc *ec2.Client) (string, error) {
	input := &ec2.DescribeClientVpnEndpointsInput{}
	result, err := svc.DescribeClientVpnEndpoints(ctx, input)
	if err != nil {
		return "", err
	}

	clientVpnEndpoint := make([]string, len(result.ClientVpnEndpoints))
	for i, vpnEndpoint := range result.ClientVpnEndpoints {
		clientVpnEndpoint[i] = *vpnEndpoint.ClientVpnEndpointId
	}

	clientVpnEndpointsJSON, err := json.Marshal(clientVpnEndpoint)
	if err != nil {
		return "", err
	}

	return string(clientVpnEndpointsJSON), nil
}

func GetIPsFromDomain(domain string) []string {
	// retrieve A/AAAA records
	log.Println("DEBUG: Fetching A/AAA IP addresses for ", domain)
	hostRecords, err := net.LookupHost(domain)
	if err == nil {
		fmt.Println("DEBUG: IP addresses:")
		for _, record := range hostRecords {
			fmt.Println(record)
		}
	} else {
		fmt.Println("Error:", err)
	}
	return hostRecords
}

// Returns not matched IPs
func GetUnmatchedIps(currentIP []string, todayIP []string) []string {
	var ips []string
	for _, today := range todayIP {
		found := false
		if !strings.Contains(today, "/32") {
			today = today + "/32"
		}
		for _, curr := range currentIP {
			if !strings.Contains(curr, "/32") {
				curr = curr + "/32"
			}
			if today == curr {
				found = true
				break
			}
		}
		if !found {
			ips = append(ips, today)
		}
	}
	return ips
}

// func HandleRequest(ctx context.Context, name MyEvent) (Response, error) {
func HandleRequest() (Response, error) {
	ctx, client, err := NewEC2Session("ap-southeast-2")

	//lukeIps := GetIPsFromDomain("api.luke.kubernetes.hipagesgroup.com.au")

	clientVpnEndpointID, err := GetVPNEndpointID(ctx, client)
	if err != nil {
		return Response{}, err
	}

	routeTables, err := GetLukeRouteTables(client, "cvpn-endpoint-0180bd612766c9023")
	if err != nil {
		return Response{}, err
	}

	ipsToAdd := GetUnmatchedIps(routeTables, GetIPsFromDomain("api.luke.kubernetes.hipagesgroup.com.au"))

	if len(ipsToAdd) == 0 {
		log.Println("All IPs matched, no changes.")
		return Response{
			Message: fmt.Sprintf("Client VPN Endpoints: %s", clientVpnEndpointID),
		}, nil
	} else {
		log.Println("IPs to add: ", ipsToAdd)
		DeleteRouteTable(ctx, client, "cvpn-endpoint-0180bd612766c9023")
		CreateRouteTable(ctx, client, "cvpn-endpoint-0180bd612766c9023", ipsToAdd, "subnet-f126ac98")
		return Response{
			Message: fmt.Sprintf("Route tables were updated in %s", clientVpnEndpointID),
		}, nil
	}
}

func main() {
	//lambda.Start(HandleRequest)
	res, err := HandleRequest()
	if err != nil {
		return
	}
	log.Println(res)
}
