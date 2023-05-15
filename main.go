package main

import (
	"log"
	"os"
)

type Response struct {
	Message string `json:"message"`
}

type Input struct {
	Domains []string `json:"domains"`
	Subnet  string   `json:"subnet"`
}

// func HandleRequest(ctx context.Context, input Input) (Response, error) {
func HandleRequest(input Input) (Response, error) {
	ctx, client, err := NewEC2Session("ap-southeast-2")

	//clientVpnEndpointID, err := GetVPNEndpointID(ctx, client)
	clientVpnEndpointID := "cvpn-endpoint-0180bd612766c9023"
	if err != nil {
		return Response{}, err
	}

	for _, domain := range input.Domains {
		ipsFromDomain := GetIPsFromDomain(domain)
		routeTables, err := GetRouteTables(client, clientVpnEndpointID, domain)
		if err != nil {
			log.Printf("ERROR: Error getting route table for %v: \n %v", clientVpnEndpointID, err)
			os.Exit(1)
		}

		if len(routeTables) == 0 {
			for _, ip := range ipsFromDomain {
				CreateRouteTable(ctx, client, clientVpnEndpointID, ip, input.Subnet, domain)
			}
		} else {
			UpdateRouteTables(ctx, client, clientVpnEndpointID, domain)
		}

		authRules, err := GetAuthorizationRules(client, clientVpnEndpointID, domain)
		if err != nil {
			log.Printf("ERROR: Error getting authorisation rules for %v: \n %v", clientVpnEndpointID, err)
			os.Exit(1)
		}

		if len(authRules) == 0 {
			for _, ip := range ipsFromDomain {
				CreateAuthorizationRules(ctx, client, clientVpnEndpointID, ip, domain)
			}
		} else {
			UpdateAuthorizationRules(ctx, client, clientVpnEndpointID, domain)
		}
	}

	return Response{
		Message: "success",
	}, nil
}

func main() {
	//lambda.Start(HandleRequest)
	domains := []string{"api.luke.kubernetes.hipagesgroup.com.au", "api.internal.luke.kubernetes.hipagesgroup.com.au"}
	res, err := HandleRequest(Input{
		Domains: domains,
		Subnet:  "subnet-f126ac98",
	})
	if err != nil {
		return
	}
	log.Println(res)
}
