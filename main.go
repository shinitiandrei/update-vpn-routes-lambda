package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"log"
	"net"
	"os"
	"strconv"
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

	clientVpnEndpoints := make([]string, len(result.ClientVpnEndpoints))
	for i, vpnEndpoint := range result.ClientVpnEndpoints {
		clientVpnEndpoints[i] = *vpnEndpoint.ClientVpnEndpointId
	}

	clientVpnEndpointsJSON, err := json.Marshal(clientVpnEndpoints)
	if err != nil {
		return "", err
	}

	return string(clientVpnEndpointsJSON), nil
}

func GetIPsFromDomain(domain string) []string {
	// retrieve A/AAAA records
	log.Println("Fetching A/AAA IP addresses for ", domain)
	hostRecords, err := net.LookupHost(domain)
	if err == nil {
		fmt.Println("IP addresses:")
		for _, record := range hostRecords {
			fmt.Println(record)
		}
	} else {
		fmt.Println("Error:", err)
	}
	return hostRecords
}

func GetLukeRouteTables(client *ec2.Client, vpnEndpointID string) ([]string, error) {
	params := &ec2.DescribeClientVpnRoutesInput{
		ClientVpnEndpointId: aws.String(vpnEndpointID),
	}

	// fetch all VPN routes using pagination
	var allRoutes []types.ClientVpnRoute

	// store IPs from Luke's load balancer
	var ips []string

	paginator := ec2.NewDescribeClientVpnRoutesPaginator(client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			log.Println("Error describing VPN routes:", err)
			os.Exit(1)
		}

		for _, route := range page.Routes {
			if route.Description != nil {
				if strings.Contains(*route.Description, "Luke API IP") {
					allRoutes = append(allRoutes, route)
					ips = append(ips, *route.DestinationCidr)
				}
			}
		}
	}

	// print the CIDR blocks of each route
	for _, route := range allRoutes {
		fmt.Println(*route.DestinationCidr)
	}

	return ips, nil
}

func CompareIPList(original []string, changed []string) bool {
	found := true
	for _, s1 := range original {
		found = false
		for _, s2 := range changed {
			if s1 == s2 {
				found = true
				break
			}
		}
		if found {
			fmt.Printf("%s found in both lists\n", s1)
		} else {
			fmt.Printf("%s not found in both lists\n", s1)
		}
	}
	return found
}

//func updateRouteTables(client *ec2.Client, vpnEndpointID, routeTableID string) error {
//	// Describe existing associations
//	associationsInput := &ec2.DescribeClientVpnTargetNetworksInput{
//		ClientVpnEndpointId: &vpnEndpointID,
//	}
//
//	associationsOutput, err := client.DescribeClientVpnTargetNetworks(client, associationsInput)
//	if err != nil {
//		return fmt.Errorf("failed to describe client VPN target networks: %v", err)
//	}
//
//	for _, association := range associationsOutput.ClientVpnTargetNetworks {
//		// Disassociate existing route table
//		disassociateInput := &ec2.DisassociateClientVpnTargetNetworkInput{
//			AssociationId:       association.AssociationId,
//			ClientVpnEndpointId: &vpnEndpointID,
//		}
//
//		_, err = client.DisassociateClientVpnTargetNetwork(ctx, disassociateInput)
//		if err != nil {
//			return fmt.Errorf("failed to disassociate route table: %v", err)
//		}
//
//		// Associate new route table
//		associateInput := &ec2.AssociateClientVpnTargetNetworkInput{
//			ClientVpnEndpointId: &vpnEndpointID,
//			SubnetId:            association.TargetNetworkId,
//		}
//
//		_, err = svc.AssociateClientVpnTargetNetwork(ctx, associateInput)
//		if err != nil {
//			return fmt.Errorf("failed to associate new route table: %v", err)
//		}
//	}
//
//	return nil
//}

// func HandleRequest(ctx context.Context) (Response, error) {
func HandleRequest() (Response, error) {
	ctx, svc, err := NewEC2Session("ap-southeast-2")

	//lukeIps := GetIPsFromDomain("api.luke.kubernetes.hipagesgroup.com.au")

	clientVpnEndpoints, err := GetVPNEndpointID(ctx, svc)
	if err != nil {
		return Response{}, err
	}

	routeTables, err := GetLukeRouteTables(svc, "cvpn-endpoint-0180bd612766c9023")
	if err != nil {
		return Response{}, err
	}

	found := CompareIPList(routeTables, GetIPsFromDomain("api.luke.kubernetes.hipagesgroup.com.au"))
	log.Println("IPs are the same? ", strconv.FormatBool(found))
	if found {
		return Response{
			Message: fmt.Sprintf("Client VPN Endpoints: %s", clientVpnEndpoints),
		}, nil
	} else {

		return Response{
			Message: fmt.Sprintf("Route tables were updated in %s", clientVpnEndpoints),
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
