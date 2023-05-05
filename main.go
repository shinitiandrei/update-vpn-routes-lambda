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

func ListClientVpnEndpoints() (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-southeast-2")) // Change to your desired region
	if err != nil {
		return "", err
	}

	svc := ec2.NewFromConfig(cfg)

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

// func HandleRequest(ctx context.Context) (Response, error) {
func HandleRequest() (Response, error) {
	clientVpnEndpoints, err := ListClientVpnEndpoints()
	if err != nil {
		return Response{}, err
	}
	log.Printf("Client VPN Endpoints: %s\n", clientVpnEndpoints)

	return Response{
		Message: fmt.Sprintf("Client VPN Endpoints: %s", clientVpnEndpoints),
	}, nil
}

func main() {
	//lambda.Start(HandleRequest)
	_, err := HandleRequest()
	if err != nil {
		return
	}
	log.Println("working")
}
