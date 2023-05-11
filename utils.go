package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"log"
	"net"
	"strings"
)

func FormatIPWith32Cidr(ip string) string {
	if !strings.Contains(ip, "/32") {
		return ip + "/32"
	} else {
		return ip
	}
}

// GetUnmatchedIPs Returns not matched IPs given 2 arrays of IPs
func GetUnmatchedIPs(original []string, toMatch []string) []string {
	var ips []string
	for _, tm := range toMatch {
		found := false
		if !strings.Contains(tm, "/32") {
			tm = tm + "/32"
		}
		for _, orig := range original {
			if !strings.Contains(orig, "/32") {
				orig = orig + "/32"
			}
			if tm == orig {
				found = true
				break
			}
		}
		if !found {
			ips = append(ips, tm)
		}
	}
	return ips
}

func GetIPsFromDomain(domain string) []string {
	// retrieve A/AAAA records
	log.Println("DEBUG: Fetching A/AAA IP addresses for ", domain)
	hostRecords, err := net.LookupHost(domain)
	if err == nil {
		fmt.Println("DEBUG: IP addresses:")
		for _, record := range hostRecords {
			fmt.Println(record)
		}
	} else {
		fmt.Println("Error:", err)
	}
	return hostRecords
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

	clientVpnEndpoint := make([]string, len(result.ClientVpnEndpoints))
	for i, vpnEndpoint := range result.ClientVpnEndpoints {
		clientVpnEndpoint[i] = *vpnEndpoint.ClientVpnEndpointId
	}

	clientVpnEndpointsJSON, err := json.Marshal(clientVpnEndpoint)
	if err != nil {
		return "", err
	}

	return string(clientVpnEndpointsJSON), nil
}
