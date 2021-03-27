package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	tfec2 "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/ec2"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/ec2/finder"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

var routeValidDestinations = []string{
	"destination_cidr_block",
	"destination_ipv6_cidr_block",
}

var routeValidTargets = []string{
	"carrier_gateway_id",
	"egress_only_gateway_id",
	"gateway_id",
	"instance_id",
	"local_gateway_id",
	"nat_gateway_id",
	"network_interface_id",
	"transit_gateway_id",
	"vpc_endpoint_id",
	"vpc_peering_connection_id",
}

func resourceAwsRoute() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRouteCreate,
		Read:   resourceAwsRouteRead,
		Update: resourceAwsRouteUpdate,
		Delete: resourceAwsRouteDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsRouteImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(2 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"route_table_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			///
			// Destinations.
			///
			"destination_cidr_block": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.Any(
					validation.StringIsEmpty,
					validateIpv4CIDRNetworkAddress,
				),
			},

			"destination_ipv6_cidr_block": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.Any(
					validation.StringIsEmpty,
					validateIpv6CIDRNetworkAddress,
				),
				DiffSuppressFunc: suppressEqualCIDRBlockDiffs,
			},

			// TODO This is a target once we have Managed Prefix List support.
			"destination_prefix_list_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			//
			// Targets.
			//
			"carrier_gateway_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ExactlyOneOf:  routeValidTargets,
				ConflictsWith: []string{"destination_ipv6_cidr_block"}, // IPv4 destinations only.
			},

			"egress_only_gateway_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ExactlyOneOf:  routeValidTargets,
				ConflictsWith: []string{"destination_cidr_block"}, // IPv6 destinations only.
			},

			"gateway_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: routeValidTargets,
			},

			"instance_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: routeValidTargets,
			},

			"local_gateway_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: routeValidTargets,
			},

			"nat_gateway_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ExactlyOneOf:  routeValidTargets,
				ConflictsWith: []string{"destination_ipv6_cidr_block"}, // IPv4 destinations only.
			},

			"network_interface_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: routeValidTargets,
			},

			"transit_gateway_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: routeValidTargets,
			},

			"vpc_endpoint_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ExactlyOneOf:  routeValidTargets,
				ConflictsWith: []string{"destination_ipv6_cidr_block"}, // IPv4 destinations only.
			},

			"vpc_peering_connection_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: routeValidTargets,
			},

			//
			// Computed attributes.
			//
			"instance_owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"origin": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsRouteCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	destinationAttributeKey, destination, err := routeDestinationAttribute(d)

	if err != nil {
		return fmt.Errorf("error creating Route: %w", err)
	}

	targetAttributeKey, target, err := routeTargetAttribute(d)

	if err != nil {
		return fmt.Errorf("error creating Route: %w", err)
	}

	routeTableID := d.Get("route_table_id").(string)
	input := &ec2.CreateRouteInput{
		RouteTableId: aws.String(routeTableID),
	}

	var routeFinder finder.RouteFinder

	switch destination := aws.String(destination); destinationAttributeKey {
	case "destination_cidr_block":
		input.DestinationCidrBlock = destination
		routeFinder = finder.RouteByIPv4Destination
	case "destination_ipv6_cidr_block":
		input.DestinationIpv6CidrBlock = destination
		routeFinder = finder.RouteByIPv6Destination
	default:
		return fmt.Errorf("error creating Route: unexpected route destination attribute: %q", destinationAttributeKey)
	}

	switch target := aws.String(target); targetAttributeKey {
	case "carrier_gateway_id":
		input.CarrierGatewayId = target
	case "egress_only_gateway_id":
		input.EgressOnlyInternetGatewayId = target
	case "gateway_id":
		input.GatewayId = target
	case "instance_id":
		input.InstanceId = target
	case "local_gateway_id":
		input.LocalGatewayId = target
	case "nat_gateway_id":
		input.NatGatewayId = target
	case "network_interface_id":
		input.NetworkInterfaceId = target
	case "transit_gateway_id":
		input.TransitGatewayId = target
	case "vpc_endpoint_id":
		input.VpcEndpointId = target
	case "vpc_peering_connection_id":
		input.VpcPeeringConnectionId = target
	default:
		return fmt.Errorf("error creating Route: unexpected route target attribute: %q", targetAttributeKey)
	}

	log.Printf("[DEBUG] Creating Route: %s", input)
	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		_, err = conn.CreateRoute(input)

		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidParameterException) {
			return resource.RetryableError(err)
		}

		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidTransitGatewayIDNotFound) {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if tfresource.TimedOut(err) {
		_, err = conn.CreateRoute(input)
	}

	if err != nil {
		return fmt.Errorf("error creating Route for Route Table (%s) with destination (%s): %w", routeTableID, destination, err)
	}

	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		_, err = routeFinder(conn, routeTableID, destination)

		if err != nil {
			return resource.RetryableError(err)
		}

		if tfresource.NotFound(err) {
			return resource.RetryableError(fmt.Errorf("route not found"))
		}

		return nil
	})

	if tfresource.TimedOut(err) {
		_, err = routeFinder(conn, routeTableID, destination)
	}

	if err != nil {
		return fmt.Errorf("error reading Route for Route Table (%s) with destination (%s): %w", routeTableID, destination, err)
	}

	if tfresource.NotFound(err) {
		return fmt.Errorf("Route in Route Table (%s) with destination (%s) not found", routeTableID, destination)
	}

	d.SetId(tfec2.RouteCreateID(routeTableID, destination))

	return resourceAwsRouteRead(d, meta)
}

func resourceAwsRouteRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	destinationAttributeKey, destination, err := routeDestinationAttribute(d)

	if err != nil {
		return fmt.Errorf("error reading Route: %w", err)
	}

	var routeFinder finder.RouteFinder

	switch destinationAttributeKey {
	case "destination_cidr_block":
		routeFinder = finder.RouteByIPv4Destination
	case "destination_ipv6_cidr_block":
		routeFinder = finder.RouteByIPv6Destination
	default:
		return fmt.Errorf("error reading Route: unexpected route destination attribute: %q", destinationAttributeKey)
	}

	routeTableID := d.Get("route_table_id").(string)

	route, err := routeFinder(conn, routeTableID, destination)

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] Route in Route Table (%s) with destination (%s) not found, removing from state", routeTableID, destination)
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading Route for Route Table (%s) with destination (%s): %w", routeTableID, destination, err)
	}

	d.Set("carrier_gateway_id", route.CarrierGatewayId)
	d.Set("destination_cidr_block", route.DestinationCidrBlock)
	d.Set("destination_ipv6_cidr_block", route.DestinationIpv6CidrBlock)
	d.Set("destination_prefix_list_id", route.DestinationPrefixListId)
	// VPC Endpoint ID is returned in Gateway ID field
	if strings.HasPrefix(aws.StringValue(route.GatewayId), "vpce-") {
		d.Set("gateway_id", "")
		d.Set("vpc_endpoint_id", route.GatewayId)
	} else {
		d.Set("gateway_id", route.GatewayId)
		d.Set("vpc_endpoint_id", "")
	}
	d.Set("egress_only_gateway_id", route.EgressOnlyInternetGatewayId)
	d.Set("nat_gateway_id", route.NatGatewayId)
	d.Set("local_gateway_id", route.LocalGatewayId)
	d.Set("instance_id", route.InstanceId)
	d.Set("instance_owner_id", route.InstanceOwnerId)
	d.Set("network_interface_id", route.NetworkInterfaceId)
	d.Set("origin", route.Origin)
	d.Set("state", route.State)
	d.Set("transit_gateway_id", route.TransitGatewayId)
	d.Set("vpc_peering_connection_id", route.VpcPeeringConnectionId)

	return nil
}

func resourceAwsRouteUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	destinationAttributeKey, destination, err := routeDestinationAttribute(d)

	if err != nil {
		return fmt.Errorf("error updating Route: %w", err)
	}

	targetAttributeKey, target, err := routeTargetAttribute(d)

	if err != nil {
		return fmt.Errorf("error updating Route: %w", err)
	}

	routeTableID := d.Get("route_table_id").(string)
	input := &ec2.ReplaceRouteInput{
		RouteTableId: aws.String(routeTableID),
	}

	switch destination := aws.String(destination); destinationAttributeKey {
	case "destination_cidr_block":
		input.DestinationCidrBlock = destination
	case "destination_ipv6_cidr_block":
		input.DestinationIpv6CidrBlock = destination
	default:
		return fmt.Errorf("error updating Route: unexpected route destination attribute: %q", destinationAttributeKey)
	}

	switch target := aws.String(target); targetAttributeKey {
	case "carrier_gateway_id":
		input.CarrierGatewayId = target
	case "egress_only_gateway_id":
		input.EgressOnlyInternetGatewayId = target
	case "gateway_id":
		input.GatewayId = target
	case "instance_id":
		input.InstanceId = target
	case "local_gateway_id":
		input.LocalGatewayId = target
	case "nat_gateway_id":
		input.NatGatewayId = target
	case "network_interface_id":
		input.NetworkInterfaceId = target
	case "transit_gateway_id":
		input.TransitGatewayId = target
	case "vpc_endpoint_id":
		input.VpcEndpointId = target
	case "vpc_peering_connection_id":
		input.VpcPeeringConnectionId = target
	default:
		return fmt.Errorf("error updating Route: unexpected route target attribute: %q", targetAttributeKey)
	}

	log.Printf("[DEBUG] Updating Route: %s", input)
	_, err = conn.ReplaceRoute(input)

	if err != nil {
		return fmt.Errorf("error updating Route for Route Table (%s) with destination (%s): %w", routeTableID, destination, err)
	}

	return resourceAwsRouteRead(d, meta)
}

func resourceAwsRouteDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	destinationAttributeKey, destination, err := routeDestinationAttribute(d)

	if err != nil {
		return fmt.Errorf("error deleting Route: %w", err)
	}

	routeTableID := d.Get("route_table_id").(string)
	input := &ec2.DeleteRouteInput{
		RouteTableId: aws.String(routeTableID),
	}

	switch destination := aws.String(destination); destinationAttributeKey {
	case "destination_cidr_block":
		input.DestinationCidrBlock = destination
	case "destination_ipv6_cidr_block":
		input.DestinationIpv6CidrBlock = destination
	default:
		return fmt.Errorf("error deleting Route: unexpected route destination attribute: %q", destinationAttributeKey)
	}

	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		log.Printf("[DEBUG] Deleting Route (%s)", d.Id())
		_, err = conn.DeleteRoute(input)
		if err == nil {
			return nil
		}

		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidRouteNotFound) {
			return nil
		}

		// Local routes (which may have been imported) cannot be deleted. Remove from state.
		if tfawserr.ErrMessageContains(err, tfec2.ErrCodeInvalidParameterValue, "cannot remove local route") {
			return nil
		}

		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidParameterException) {
			return resource.RetryableError(err)
		}

		return resource.NonRetryableError(err)
	})

	if tfresource.TimedOut(err) {
		log.Printf("[DEBUG] Deleting Route (%s)", d.Id())
		_, err = conn.DeleteRoute(input)
	}

	if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidRouteNotFound) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting Route for Route Table (%s) with destination (%s): %w", routeTableID, destination, err)
	}

	return nil
}

func resourceAwsRouteImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), "_")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		return nil, fmt.Errorf("unexpected format of ID (%q), expected ROUTETABLEID_DESTINATION", d.Id())
	}

	routeTableID := idParts[0]
	destination := idParts[1]
	d.Set("route_table_id", routeTableID)
	if strings.Contains(destination, ":") {
		d.Set("destination_ipv6_cidr_block", destination)
	} else {
		d.Set("destination_cidr_block", destination)
	}
	d.SetId(tfec2.RouteCreateID(routeTableID, destination))

	return []*schema.ResourceData{d}, nil
}

// routeDestinationAttribute returns the attribute key and value of the route's destination.
func routeDestinationAttribute(d *schema.ResourceData) (string, string, error) {
	for _, key := range routeValidDestinations {
		if v, ok := d.Get(key).(string); ok && v != "" {
			return key, v, nil
		}
	}

	return "", "", fmt.Errorf("route destination attribute not specified")
}

// routeTargetAttribute returns the attribute key and value of the route's target.
func routeTargetAttribute(d *schema.ResourceData) (string, string, error) {
	for _, key := range routeValidTargets {
		// The HasChange check is necessary to handle Computed attributes that will be cleared once they are read back after update.
		if v, ok := d.Get(key).(string); ok && v != "" && d.HasChange(key) {
			return key, v, nil
		}
	}

	return "", "", fmt.Errorf("route target attribute not specified")
}
