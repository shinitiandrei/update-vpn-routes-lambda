package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"log"
	"os"
	"strings"
)

func DeleteRouteTable(ctx context.Context, client *ec2.Client, vpnEndpointID string, ip string, subnetID string) {
	ipFormatted := FormatIPWith32Cidr(ip)

	_, err := client.DeleteClientVpnRoute(ctx, &ec2.DeleteClientVpnRouteInput{
		ClientVpnEndpointId:  &vpnEndpointID,
		DestinationCidrBlock: &ipFormatted,
		TargetVpcSubnetId:    &subnetID,
	})

	if err != nil {
		log.Printf("ERROR: Error deleting route table for %v: \n %v", ip, err)
		os.Exit(1)
	} else {
		log.Printf("INFO: Route table deleted: %v", ip)
	}
}

func CreateRouteTable(ctx context.Context, client *ec2.Client, vpnEndpointID string, ip string, subnetId string, desc string) {
	ipFormatted := FormatIPWith32Cidr(ip)
	_, err := client.CreateClientVpnRoute(ctx, &ec2.CreateClientVpnRouteInput{
		ClientVpnEndpointId:  &vpnEndpointID,
		DestinationCidrBlock: &ipFormatted,
		TargetVpcSubnetId:    &subnetId,
		Description:          &desc,
	})

	if err != nil {
		log.Printf("ERROR: Error creating VPN route %v: \n %v", ip, err)
		os.Exit(1)
	} else {
		log.Printf("INFO: Route table created: %v", ip)
	}
}

func GetRouteTables(client *ec2.Client, vpnEndpointID string, desc string) ([]types.ClientVpnRoute, error) {
	params := &ec2.DescribeClientVpnRoutesInput{
		ClientVpnEndpointId: aws.String(vpnEndpointID),
	}

	var allRoutes []types.ClientVpnRoute

	paginator := ec2.NewDescribeClientVpnRoutesPaginator(client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			log.Println("Error describing VPN routes:", err)
			os.Exit(1)
		}

		for _, route := range page.Routes {
			if route.Description != nil {
				if strings.Contains(*route.Description, desc) {
					allRoutes = append(allRoutes, route)
				}
			}
		}
	}

	return allRoutes, nil
}

func UpdateRouteTables(ctx context.Context, client *ec2.Client, clientVpnEndpointID string, domain string) {
	routeTables, err := GetRouteTables(client, clientVpnEndpointID, domain)
	ipsFromDomain := GetIPsFromDomain(domain)
	if err != nil {
		log.Printf("Error getting route tables from VPN client %v: \n %v", clientVpnEndpointID, err)
		os.Exit(1)
	}

	var routeTableDestCidrs []string
	for _, rt := range routeTables {
		routeTableDestCidrs = append(routeTableDestCidrs, *rt.DestinationCidr)
	}

	ipsToAdd := GetUnmatchedIPs(routeTableDestCidrs, ipsFromDomain)
	ipsToRemove := GetUnmatchedIPs(ipsFromDomain, routeTableDestCidrs)

	if len(ipsToAdd) == 0 {
		log.Println("INFO: All IPs matched in route tables, no changes applied.")
	} else {
		// Stores the description that's about to be replaced
		var descToReplace []string
		var subnet string

		// It will match the ip to be removed and store its description to be added as a new IP.
		for _, ip := range ipsToRemove {
			for _, rt := range routeTables {
				if ip == *rt.DestinationCidr {
					descToReplace = append(descToReplace, *rt.Description)
					subnet = *rt.TargetSubnet
					DeleteRouteTable(ctx, client, clientVpnEndpointID, ip, subnet)
				}
			}
		}

		log.Printf("Description to be replaced: %v", descToReplace)
		log.Printf("IPs to be removed: %v", ipsToRemove)
		log.Printf("IPs to be added: %v", ipsToAdd)

		for _, desc := range descToReplace {
			for _, ip := range ipsToAdd {
				CreateRouteTable(ctx, client, clientVpnEndpointID, ip, subnet, desc)
				log.Printf("INFO: Route tables were updated in %s\n", clientVpnEndpointID)
			}
		}
	}
}
