package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"log"
	"os"
	"strings"
)

type ClientVpnAuthorizationRule struct {
	DestCidr    *string
	Description *string
}

func DeleteAuthorizationRules(ctx context.Context, client *ec2.Client, vpnEndpointID string, ip string) {
	ipFormatted := FormatIPWith32Cidr(ip)
	revokeAllGroups := true

	_, err := client.RevokeClientVpnIngress(ctx, &ec2.RevokeClientVpnIngressInput{
		ClientVpnEndpointId: &vpnEndpointID,
		TargetNetworkCidr:   &ipFormatted,
		RevokeAllGroups:     &revokeAllGroups,
	})

	if err != nil {
		log.Printf("ERROR: Error deleting auhtorization rule for %v: \n %v", ip, err)
		os.Exit(1)
	} else {
		log.Printf("INFO: Authorization rule deleted for: %v", ip)
	}
}

func CreateAuthorizationRules(ctx context.Context, client *ec2.Client, vpnEndpointID string, ip string, desc string) {
	ipFormatted := FormatIPWith32Cidr(ip)
	authorizeAllGroups := true

	_, err := client.AuthorizeClientVpnIngress(ctx, &ec2.AuthorizeClientVpnIngressInput{
		ClientVpnEndpointId: &vpnEndpointID,
		Description:         &desc,
		TargetNetworkCidr:   &ipFormatted,
		AuthorizeAllGroups:  &authorizeAllGroups,
	})

	if err != nil {
		log.Printf("ERROR: Error creating auhtorization rule for %v: \n %v", ip, err)
		os.Exit(1)
	} else {
		log.Printf("Authorization rule created for: %v", ip)
	}
}

func GetAuthorizationRules(client *ec2.Client, vpnEndpointID string) ([]ClientVpnAuthorizationRule, error) {
	params := &ec2.DescribeClientVpnAuthorizationRulesInput{
		ClientVpnEndpointId: aws.String(vpnEndpointID),
	}

	var allAuthRules []ClientVpnAuthorizationRule

	// fetch all VPN Authorization rules using pagination
	paginator := ec2.NewDescribeClientVpnAuthorizationRulesPaginator(client, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			log.Println("Error describing VPN authorization rules:", err)
			os.Exit(1)
		}

		for _, authrule := range page.AuthorizationRules {
			if authrule.Description != nil {
				if strings.Contains(*authrule.Description, "Luke Test") {
					allAuthRules = append(allAuthRules, ClientVpnAuthorizationRule{
						Description: authrule.Description,
						DestCidr:    authrule.DestinationCidr,
					})
				}
			}
		}
	}
	return allAuthRules, nil
}

func UpdateAuthorizationRules(ctx context.Context, client *ec2.Client, clientVpnEndpointID string, domain string) {
	authRules, err := GetAuthorizationRules(client, clientVpnEndpointID)
	if err != nil {
		log.Printf("Error getting auth rules from AWS VPN ID %v: \n %v", clientVpnEndpointID, err)
		os.Exit(1)
	}

	var authDestCidrs []string
	for _, authRule := range authRules {
		authDestCidrs = append(authDestCidrs, *authRule.DestCidr)
	}

	ipsToAdd := GetUnmatchedIPs(authDestCidrs, GetIPsFromDomain(domain))
	ipsToRemove := GetUnmatchedIPs(GetIPsFromDomain(domain), authDestCidrs)

	if len(ipsToAdd) == 0 {
		log.Println("INFO: All IPs match in authorization rules, no changes.")
	} else {

		// Stores the description that's about to be replaced
		var descToReplace []string

		// It will match the ip to be removed and store its description to be added as a new IP.
		for _, ip := range ipsToRemove {
			for _, rule := range authRules {
				if ip == *rule.DestCidr {
					descToReplace = append(descToReplace, *rule.Description)
					DeleteAuthorizationRules(ctx, client, clientVpnEndpointID, ip)
				}
			}
		}

		log.Printf("Description to be replaced: %v", descToReplace)
		log.Printf("IPs to be removed: %v", ipsToRemove)
		log.Printf("IPs to be added: %v", ipsToAdd)

		if len(descToReplace) == len(ipsToAdd) {
			for _, desc := range descToReplace {
				for _, ip := range ipsToAdd {
					log.Println("INFO: IPs to add: ", ipsToAdd)
					log.Println("INFO: IPs to remove: ", ipsToRemove)
					CreateAuthorizationRules(ctx, client, clientVpnEndpointID, ip, desc)
					log.Printf("Authorization rules were updated in %s\n", clientVpnEndpointID, ip)
				}
			}
		} else {
			log.Printf("ERROR: number(%v) of ips don't match the number(%v) of descriptions", len(ipsToAdd), len(descToReplace))
			os.Exit(1)
		}
	}
}
