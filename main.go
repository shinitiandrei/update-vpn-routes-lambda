package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"log"
)

type Response struct {
	Message string `json:"message"`
}

func NewAWSSession(region string) (context.Context, *ec2.Client, error) {
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
	result, err := svc.DescribeClientVpnEndpoints(context.TODO(), input)
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
	targetNetworksInput := &ec2.DescribeClientVpnTargetNetworksInput{
		ClientVpnEndpointId: &clientVpnEndpointId,
	}
	targetNetworksResult, err := svc.DescribeClientVpnTargetNetworks(context.TODO(), targetNetworksInput)
	if err != nil {
		return nil, err
	}

	associatedSubnets := make(map[string]struct{}, len(targetNetworksResult.ClientVpnTargetNetworks))
	for _, targetNetwork := range targetNetworksResult.ClientVpnTargetNetworks {
		associatedSubnets[*targetNetwork.TargetNetworkId] = struct{}{}
	}

	routeTablesInput := &ec2.DescribeRouteTablesInput{}
	routeTablesResult, err := svc.DescribeRouteTables(context.TODO(), routeTablesInput)
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

// func HandleRequest(ctx context.Context) (Response, error) {
func HandleRequest() (Response, error) {
	ctx, svc, err := NewAWSSession("ap-southeast-2")

	clientVpnEndpoints, err := ListClientVpnEndpoints(ctx, svc)
	if err != nil {
		return Response{}, err
	}

	routeTables, err := GetAssociatedRouteTables(ctx, svc, clientVpnEndpoints)

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
