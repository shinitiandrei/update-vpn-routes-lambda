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

func ListClientVpnEndpoints(ctx context.Context, svc *ec2.Client) (string, error) {
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

func GetAssociatedRouteTables(ctx context.Context, svc *ec2.Client, clientVpnEndpointId string) ([]string, error) {

	sample := &ec2.DescribeClientVpnRoutesInput{
		ClientVpnEndpointId: &clientVpnEndpointId,
	}

	log.Print(sample.MaxResults)

	targetNetworksInput := &ec2.DescribeClientVpnTargetNetworksInput{
		ClientVpnEndpointId: &clientVpnEndpointId,
	}

	log.Print(clientVpnEndpointId)

	targetNetworksResult, err := svc.DescribeClientVpnTargetNetworks(ctx, targetNetworksInput)
	log.Println(targetNetworksResult)
	if err != nil {
		return nil, err
	}

	associatedSubnets := make(map[string]struct{}, len(targetNetworksResult.ClientVpnTargetNetworks))
	for _, targetNetwork := range targetNetworksResult.ClientVpnTargetNetworks {
		associatedSubnets[*targetNetwork.TargetNetworkId] = struct{}{}
	}

	routeTablesInput := &ec2.DescribeRouteTablesInput{}
	routeTablesResult, err := svc.DescribeRouteTables(ctx, routeTablesInput)
	if err != nil {
		return nil, err
	}

	associatedRouteTables := make([]string, 0)
	for _, routeTable := range routeTablesResult.RouteTables {
		for _, association := range routeTable.Associations {
			if _, ok := associatedSubnets[*association.SubnetId]; ok {
				associatedRouteTables = append(associatedRouteTables, *routeTable.RouteTableId)
				break
			}
		}
	}

	return associatedRouteTables, nil
}

func getRouteTables(ctx context.Context, svc *ec2.Client, vpnEndpointID string) ([]*types.RouteTable, error) {
	params := &ec2.DescribeClientVpnRoutesInput{
		ClientVpnEndpointId: aws.String(vpnEndpointID),
	}

	result, err := svc.DescribeClientVpnRoutes(ctx, params)

	if err != nil {
		return nil, fmt.Errorf("failed to describe client VPN target networks: %v", err)
	}

	routeTableDesc := make([]string, 0, len(result.Routes))
	for _, route := range result.Routes {
		log.Println(*route.Description)
		log.Println(*route.DestinationCidr)
		routeTableDesc = append(routeTableDesc, *route.Description)
	}

	routeTablesInput := &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("association.route-table-id"),
				Values: routeTableDesc,
			},
		},
	}

	routeTablesOutput, err := svc.DescribeRouteTables(ctx, routeTablesInput)
	if err != nil {
		return nil, fmt.Errorf("failed to describe route tables: %v", err)
	}

	routeTables := make([]*types.RouteTable, len(routeTablesOutput.RouteTables))
	for i, rt := range routeTablesOutput.RouteTables {
		routeTables[i] = &rt
	}

	return routeTables, nil
}

// func HandleRequest(ctx context.Context) (Response, error) {
func HandleRequest() (Response, error) {
	ctx, svc, err := NewEC2Session("ap-southeast-2")

	clientVpnEndpoints, err := ListClientVpnEndpoints(ctx, svc)
	if err != nil {
		return Response{}, err
	}

	routeTables, err := getRouteTables(ctx, svc, "cvpn-endpoint-0180bd612766c9023")

	return Response{
		Message: fmt.Sprintf("Client VPN Endpoints: %s \n route tables: %s", clientVpnEndpoints, routeTables),
	}, nil
}

func main() {
	//lambda.Start(HandleRequest)
	res, err := HandleRequest()
	if err != nil {
		return
	}
	log.Println(res)
}
