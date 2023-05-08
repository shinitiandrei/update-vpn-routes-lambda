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
	"os"
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

// func getRouteTables(ctx context.Context, svc *ec2.Client, vpnEndpointID string) ([]*types.RouteTable, error) {
func getRouteTables(ctx context.Context, svc *ec2.Client, vpnEndpointID string) (string, error) {
	params := &ec2.DescribeClientVpnRoutesInput{
		ClientVpnEndpointId: aws.String(vpnEndpointID),
	}

	//result, err := svc.DescribeClientVpnRoutes(ctx, params)

	// fetch all VPN routes using pagination
	var allRoutes []*types.ClientVpnRoute
	paginator := ec2.NewDescribeClientVpnRoutesPaginator(svc, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			fmt.Println("Error describing VPN routes:", err)
			os.Exit(1)
		}

		for _, route := range page.Routes {
			allRoutes = append(allRoutes, &route)
		}
	}

	// print the CIDR blocks of each route
	for _, route := range allRoutes {
		fmt.Println(*route.Description)
	}

	//routeTableDesc := make([]string, 0, len(routes.Routes))
	//
	//routeTablesInput := &ec2.DescribeRouteTablesInput{
	//	Filters: []types.Filter{
	//		{
	//			Name:   aws.String("association.route-table-id"),
	//			Values: routeTableDesc,
	//		},
	//	},
	//}

	//routeTablesOutput, err := svc.DescribeRouteTables(ctx, routeTablesInput)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to describe route tables: %v", err)
	//}
	//
	//routeTables := make([]*types.RouteTable, len(routeTablesOutput.RouteTables))
	//for i, rt := range routeTablesOutput.RouteTables {
	//	routeTables[i] = &rt
	//}

	return "routeTables", nil
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
