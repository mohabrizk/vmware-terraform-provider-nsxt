/* Copyright © 2017 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: MPL-2.0 */

package nsxt

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	api "github.com/vmware/go-vmware-nsxt"
	"github.com/vmware/go-vmware-nsxt/manager"
	"log"
	"net/http"
)

var natRuleActionValues = []string{"SNAT", "DNAT", "NO_NAT", "REFLEXIVE"}

func resourceNsxtNatRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceNsxtNatRuleCreate,
		Read:   resourceNsxtNatRuleRead,
		Update: resourceNsxtNatRuleUpdate,
		Delete: resourceNsxtNatRuleDelete,

		Schema: map[string]*schema.Schema{
			"revision": getRevisionSchema(),
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Description of this resource",
				Optional:    true,
			},
			"display_name": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The display name of this resource. Defaults to ID if not set",
				Optional:    true,
				Computed:    true,
			},
			"tag": getTagsSchema(),
			"action": &schema.Schema{
				Type:         schema.TypeString,
				Description:  "valid actions: SNAT, DNAT, NO_NAT, REFLEXIVE. All rules in a logical router are either stateless or stateful. Mix is not supported. SNAT and DNAT are stateful, can NOT be supported when the logical router is running at active-active HA mode; REFLEXIVE is stateless. NO_NAT has no translated_fields, only match fields",
				Required:     true,
				ValidateFunc: validation.StringInSlice(natRuleActionValues, false),
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Default:     true,
				Description: "enable/disable the rule",
				Optional:    true,
			},
			"logging": &schema.Schema{
				Type:        schema.TypeBool,
				Default:     false,
				Description: "enable/disable the logging of rule",
				Optional:    true,
			},
			"logical_router_id": &schema.Schema{
				Type:        schema.TypeString,
				Description: "Logical router id",
				Required:    true,
			},
			"match_destination_network": &schema.Schema{
				Type:        schema.TypeString,
				Description: "IP Address | CIDR | (null implies Any)",
				Optional:    true,
			},
			"match_source_network": &schema.Schema{
				Type:        schema.TypeString,
				Description: "IP Address | CIDR | (null implies Any)",
				Optional:    true,
			},
			"nat_pass": &schema.Schema{
				Type:        schema.TypeBool,
				Default:     true,
				Description: "Default is true. If the natPass is set to true, the following firewall stage will be skipped. Please note, if action is NO_NAT, then natPass must be set to true or omitted",
				Optional:    true,
			},
			"rule_priority": &schema.Schema{
				Type:        schema.TypeInt,
				Description: "Ascending, valid range [0-2147483647]. If multiple rules have the same priority, evaluation sequence is undefined",
				Computed:    true,
			},
			"translated_network": &schema.Schema{
				Type:        schema.TypeString,
				Description: "IP Address | IP Range | CIDR. For DNAT rules only a single ip is supported",
				Optional:    true,
			},
			"translated_ports": &schema.Schema{
				Type:        schema.TypeString,
				Description: "port number or port range. DNAT only",
				Optional:    true,
			},
			//TODO(asarfaty): Add match_service field
		},
	}
}

func resourceNsxtNatRuleCreate(d *schema.ResourceData, m interface{}) error {
	nsxClient := m.(*api.APIClient)
	logicalRouterID := d.Get("logical_router_id").(string)
	if logicalRouterID == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	description := d.Get("description").(string)
	displayName := d.Get("display_name").(string)
	tags := getTagsFromSchema(d)
	action := d.Get("action").(string)
	enabled := d.Get("enabled").(bool)
	logging := d.Get("logging").(bool)
	matchDestinationNetwork := d.Get("match_destination_network").(string)
	//match_service := d.Get("match_service").(*NsServiceElement)
	matchSourceNetwork := d.Get("match_source_network").(string)
	natPass := d.Get("nat_pass").(bool)
	rulePriority := int64(d.Get("rule_priority").(int))
	translatedNetwork := d.Get("translated_network").(string)
	translatedPorts := d.Get("translated_ports").(string)
	natRule := manager.NatRule{
		Description:             description,
		DisplayName:             displayName,
		Tags:                    tags,
		Action:                  action,
		Enabled:                 enabled,
		Logging:                 logging,
		LogicalRouterId:         logicalRouterID,
		MatchDestinationNetwork: matchDestinationNetwork,
		//MatchService: match_service,
		MatchSourceNetwork: matchSourceNetwork,
		NatPass:            natPass,
		RulePriority:       rulePriority,
		TranslatedNetwork:  translatedNetwork,
		TranslatedPorts:    translatedPorts,
	}

	natRule, resp, err := nsxClient.LogicalRoutingAndServicesApi.AddNatRule(nsxClient.Context, logicalRouterID, natRule)

	if err != nil {
		return fmt.Errorf("Error during NatRule create: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Unexpected status returned during NatRule create: %v", resp.StatusCode)
	}
	d.SetId(natRule.Id)

	return resourceNsxtNatRuleRead(d, m)
}

func resourceNsxtNatRuleRead(d *schema.ResourceData, m interface{}) error {
	nsxClient := m.(*api.APIClient)
	id := d.Id()
	if id == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	logicalRouterID := d.Get("logical_router_id").(string)
	if logicalRouterID == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	natRule, resp, err := nsxClient.LogicalRoutingAndServicesApi.GetNatRule(nsxClient.Context, logicalRouterID, id)
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[DEBUG] NatRule %s not found", id)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error during NatRule read: %v", err)
	}

	d.Set("revision", natRule.Revision)
	d.Set("description", natRule.Description)
	d.Set("display_name", natRule.DisplayName)
	setTagsInSchema(d, natRule.Tags)
	d.Set("action", natRule.Action)
	d.Set("enabled", natRule.Enabled)
	d.Set("logging", natRule.Logging)
	d.Set("logical_router_id", natRule.LogicalRouterId)
	d.Set("match_destination_network", natRule.MatchDestinationNetwork)
	//d.Set("match_service", natRule.MatchService)
	d.Set("match_source_network", natRule.MatchSourceNetwork)
	d.Set("nat_pass", natRule.NatPass)
	d.Set("rule_priority", natRule.RulePriority)
	d.Set("translated_network", natRule.TranslatedNetwork)
	d.Set("translated_ports", natRule.TranslatedPorts)

	return nil
}

func resourceNsxtNatRuleUpdate(d *schema.ResourceData, m interface{}) error {
	nsxClient := m.(*api.APIClient)
	id := d.Id()
	if id == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	logicalRouterID := d.Get("logical_router_id").(string)
	if logicalRouterID == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	revision := int64(d.Get("revision").(int))
	description := d.Get("description").(string)
	displayName := d.Get("display_name").(string)
	tags := getTagsFromSchema(d)
	action := d.Get("action").(string)
	enabled := d.Get("enabled").(bool)
	logging := d.Get("logging").(bool)
	matchDestinationNetwork := d.Get("match_destination_network").(string)
	//match_service := d.Get("match_service").(*NsServiceElement)
	matchSourceNetwork := d.Get("match_source_network").(string)
	natPass := d.Get("nat_pass").(bool)
	rulePriority := int64(d.Get("rule_priority").(int))
	translatedNetwork := d.Get("translated_network").(string)
	translatedPorts := d.Get("translated_ports").(string)
	natRule := manager.NatRule{
		Revision:                revision,
		Description:             description,
		DisplayName:             displayName,
		Tags:                    tags,
		Action:                  action,
		Enabled:                 enabled,
		Logging:                 logging,
		LogicalRouterId:         logicalRouterID,
		MatchDestinationNetwork: matchDestinationNetwork,
		//MatchService: match_service,
		MatchSourceNetwork: matchSourceNetwork,
		NatPass:            natPass,
		RulePriority:       rulePriority,
		TranslatedNetwork:  translatedNetwork,
		TranslatedPorts:    translatedPorts,
	}

	natRule, resp, err := nsxClient.LogicalRoutingAndServicesApi.UpdateNatRule(nsxClient.Context, logicalRouterID, id, natRule)

	if err != nil || resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Error during NatRule update: %v", err)
	}

	return resourceNsxtNatRuleRead(d, m)
}

func resourceNsxtNatRuleDelete(d *schema.ResourceData, m interface{}) error {
	nsxClient := m.(*api.APIClient)
	id := d.Id()
	if id == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}
	logicalRouterID := d.Get("logical_router_id").(string)
	if logicalRouterID == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	resp, err := nsxClient.LogicalRoutingAndServicesApi.DeleteNatRule(nsxClient.Context, logicalRouterID, id)
	if err != nil {
		return fmt.Errorf("Error during NatRule delete: %v", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[DEBUG] NatRule %s not found", id)
		d.SetId("")
	}
	return nil
}