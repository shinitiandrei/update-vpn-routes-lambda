package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"log"
	"os"
	"strconv"
	"strings"
)

func DeleteAuthorizationRules(ctx context.Context, client *ec2.Client, vpnEndpointID string, ips []string) {
	params := &ec2.DescribeClientVpnRoutesInput{
		ClientVpnEndpointId: aws.String(vpnEndpointID),
	}

	paginator := ec2.NewDescribeClientVpnRoutesPaginator(client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			log.Println("Error describing VPN routes:", err)
			os.Exit(1)
		}

		for _, route := range page.Routes {
			if route.Description != nil {
				if strings.Contains(*route.Description, "Luke API IP Test") {
					log.Printf("Deleting route table: %v", *route.DestinationCidr)
					_, err = client.DeleteClientVpnRoute(ctx, &ec2.DeleteClientVpnRouteInput{
						ClientVpnEndpointId:  &vpnEndpointID,
						DestinationCidrBlock: route.DestinationCidr,
						TargetVpcSubnetId:    route.TargetSubnet,
					})
					if err != nil {
						log.Printf("Error deleting VPN route %v: \n %v", *route.DestinationCidr, err)
						os.Exit(1)
					} else {
						log.Printf("Route table deleted: %v", *route.DestinationCidr)
					}
				}
			}
		}
	}
}

func CreateAuthorizationRules(ctx context.Context, client *ec2.Client, vpnEndpointID string, ips []string, subnetId string) {
	var suffix int
	for _, ip := range ips {
		suffix = suffix + 1
		description := "Luke API IP Test" + strconv.Itoa(suffix)

		var ipFormatted string
		if !strings.Contains(ip, "/32") {
			ipFormatted = ip + "/32"
		} else {
			ipFormatted = ip
		}

		_, err := client.CreateClientVpnRoute(ctx, &ec2.CreateClientVpnRouteInput{
			ClientVpnEndpointId:  &vpnEndpointID,
			DestinationCidrBlock: &ipFormatted,
			TargetVpcSubnetId:    &subnetId,
			Description:          &description,
		})
		if err != nil {
			log.Printf("Error creating auhtorization rule for %v: \n %v", ip, err)
			os.Exit(1)
		} else {
			log.Printf("Authorization rule created for: %v", ip)
		}
	}
}

func GetLukeAuthorizationRules(client *ec2.Client, vpnEndpointID string) ([]string, error) {
	params := &ec2.DescribeClientVpnAuthorizationRulesInput{
		ClientVpnEndpointId: aws.String(vpnEndpointID),
	}

	// fetch all VPN routes using pagination
	var allRoutes []types.AuthorizationRule

	// store IPs from Luke's load balancer
	var ips []string

	paginator := ec2.NewDescribeClientVpnAuthorizationRulesPaginator(client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			log.Println("Error describing VPN routes:", err)
			os.Exit(1)
		}

		for _, authrule := range page.AuthorizationRules {
			if authrule.Description != nil {
				if strings.Contains(*authrule.Description, "Luke") {
					allRoutes = append(allRoutes, authrule)
					ips = append(ips, *authrule.DestinationCidr)
				}
			}
		}
	}
	return ips, nil
}

func UpdateAuthorizationRules(ctx context.Context, client *ec2.Client, clientVpnEndpointID string) {
	authRules, err := GetLukeAuthorizationRules(client, clientVpnEndpointID)
	if err != nil {
		log.Printf("Error getting auth rules from AWS VPN ID %v: \n %v", clientVpnEndpointID, err)
		os.Exit(1)
	}

	log.Printf("Auth rules: %v\n", authRules)

	ipsToAdd := GetUnmatchedIpsFromLookup(authRules, GetIPsFromDomain("api.luke.kubernetes.hipagesgroup.com.au"))
	ipsToRemove := GetUnmatchedIpsFromVPN(authRules, GetIPsFromDomain("api.luke.kubernetes.hipagesgroup.com.au"))

	if len(ipsToAdd) == 0 {
		log.Println("All IPs match, no changes.")
	} else {
		log.Println("IPs to add: ", ipsToAdd)
		DeleteAuthorizationRules(ctx, client, clientVpnEndpointID, ipsToRemove)
		//CreateAuthorizationRules(ctx, client, clientVpnEndpointID, ipsToAdd, "subnet-f126ac98")
		log.Printf("Authorization rules were updated in %s\n", clientVpnEndpointID)
	}
}
