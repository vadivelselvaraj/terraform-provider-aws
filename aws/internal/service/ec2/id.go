package ec2

import (
	"fmt"
	"strings"

	"github.com/terraform-providers/terraform-provider-aws/aws/internal/hashcode"
)

const clientVpnAuthorizationRuleIDSeparator = ","

func ClientVpnAuthorizationRuleCreateID(endpointID, targetNetworkCidr, accessGroupID string) string {
	parts := []string{endpointID, targetNetworkCidr}
	if accessGroupID != "" {
		parts = append(parts, accessGroupID)
	}
	id := strings.Join(parts, clientVpnAuthorizationRuleIDSeparator)
	return id
}

func ClientVpnAuthorizationRuleParseID(id string) (string, string, string, error) {
	parts := strings.Split(id, clientVpnAuthorizationRuleIDSeparator)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], "", nil
	}
	if len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != "" {
		return parts[0], parts[1], parts[2], nil
	}

	return "", "", "",
		fmt.Errorf("unexpected format for ID (%q), expected endpoint-id"+clientVpnAuthorizationRuleIDSeparator+
			"target-network-cidr or endpoint-id"+clientVpnAuthorizationRuleIDSeparator+"target-network-cidr"+
			clientVpnAuthorizationRuleIDSeparator+"group-id", id)
}

const clientVpnNetworkAssociationIDSeparator = ","

func ClientVpnNetworkAssociationCreateID(endpointID, associationID string) string {
	parts := []string{endpointID, associationID}
	id := strings.Join(parts, clientVpnNetworkAssociationIDSeparator)
	return id
}

func ClientVpnNetworkAssociationParseID(id string) (string, string, error) {
	parts := strings.Split(id, clientVpnNetworkAssociationIDSeparator)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], nil
	}

	return "", "",
		fmt.Errorf("unexpected format for ID (%q), expected endpoint-id"+clientVpnNetworkAssociationIDSeparator+
			"association-id", id)
}

const clientVpnRouteIDSeparator = ","

func ClientVpnRouteCreateID(endpointID, targetSubnetID, destinationCidr string) string {
	parts := []string{endpointID, targetSubnetID, destinationCidr}
	id := strings.Join(parts, clientVpnRouteIDSeparator)
	return id
}

func ClientVpnRouteParseID(id string) (string, string, string, error) {
	parts := strings.Split(id, clientVpnRouteIDSeparator)
	if len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != "" {
		return parts[0], parts[1], parts[2], nil
	}

	return "", "", "",
		fmt.Errorf("unexpected format for ID (%q), expected endpoint-id"+clientVpnRouteIDSeparator+
			"target-subnet-id"+clientVpnRouteIDSeparator+"destination-cidr-block", id)
}

// RouteCreateID returns a route resource ID.
func RouteCreateID(routeTableID, destination string) string {
	return fmt.Sprintf("r-%s%d", routeTableID, hashcode.String(destination))
}

const transitGatewayPrefixListReferenceSeparator = "_"

func TransitGatewayPrefixListReferenceCreateID(transitGatewayRouteTableID string, prefixListID string) string {
	parts := []string{transitGatewayRouteTableID, prefixListID}
	id := strings.Join(parts, transitGatewayPrefixListReferenceSeparator)

	return id
}

func TransitGatewayPrefixListReferenceParseID(id string) (string, string, error) {
	parts := strings.Split(id, transitGatewayPrefixListReferenceSeparator)

	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("unexpected format for ID (%[1]s), expected transit-gateway-route-table-id%[2]sprefix-list-id", id, transitGatewayPrefixListReferenceSeparator)
}

func VpnGatewayVpcAttachmentCreateID(vpnGatewayID, vpcID string) string {
	return fmt.Sprintf("vpn-attachment-%x", hashcode.String(fmt.Sprintf("%s-%s", vpcID, vpnGatewayID)))
}
