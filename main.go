package main

import (
	"log"
)

type Response struct {
	Message string `json:"message"`
}

// func HandleRequest(ctx context.Context, name MyEvent) (Response, error) {
func HandleRequest() (Response, error) {
	ctx, client, err := NewEC2Session("ap-southeast-2")

	clientVpnEndpointID, err := GetVPNEndpointID(ctx, client)
	if err != nil {
		return Response{}, err
	}
	log.Println(clientVpnEndpointID)

	//UpdateRouteTables(ctx, client, "cvpn-endpoint-0180bd612766c9023")
	UpdateAuthorizationRules(ctx, client, "cvpn-endpoint-0180bd612766c9023")

	return Response{
		Message: "success",
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
