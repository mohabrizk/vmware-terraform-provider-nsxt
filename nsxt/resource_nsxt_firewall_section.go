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

var firewallRuleIPProtocolValues = []string{"IPV4", "IPV6", "IPV4_IPV6"}
var firewallRuleActionValues = []string{"ALLOW", "DROP", "REJECT"}
var firewallRuleDirectionValues = []string{"IN", "OUT", "IN_OUT"}
var firewallSectionTypeValues = []string{"LAYER2", "LAYER3"}

func resourceNsxtFirewallSection() *schema.Resource {
	return &schema.Resource{
		Create: resourceNsxtFirewallSectionCreate,
		Read:   resourceNsxtFirewallSectionRead,
		Update: resourceNsxtFirewallSectionUpdate,
		Delete: resourceNsxtFirewallSectionDelete,

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
			"is_default": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "A boolean flag which reflects whether a firewall section is default section or not. Each Layer 3 and Layer 2 section will have at least and at most one default section",
				Computed:    true,
			},
			"section_type": &schema.Schema{
				Type:         schema.TypeString,
				Description:  "Type of the rules which a section can contain. Only homogeneous sections are supported",
				Required:     true,
				ValidateFunc: validation.StringInSlice(firewallSectionTypeValues, false),
			},
			"stateful": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "Stateful or Stateless nature of firewall section is enforced on all rules inside the section. Layer3 sections can be stateful or stateless. Layer2 sections can only be stateless",
				Required:    true,
				ForceNew:    true,
			},
			"applied_to": getResourceReferencesSetSchema(false, false, []string{"LogicalPort", "LogicalSwitch", "NSGroup"}, "List of objects where the rules in this section will be enforced. This will take precedence over rule level appliedTo"),
			"rule":       getRulesSchema(),
		},
	}
}

func getRulesSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "List of firewall rules in the section. Only homogeneous rules are supported",
		Optional:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"revision": getRevisionSchema(),
				"description": &schema.Schema{
					Type:        schema.TypeString,
					Description: "Description of this resource",
					Optional:    true,
				},
				"display_name": &schema.Schema{
					Type:        schema.TypeString,
					Description: "Defaults to ID if not set",
					Optional:    true,
				},
				"action": &schema.Schema{
					Type:         schema.TypeString,
					Description:  "Action enforced on the packets which matches the firewall rule",
					Required:     true,
					ValidateFunc: validation.StringInSlice(firewallRuleActionValues, false),
				},
				"applied_to":  getResourceReferencesSchema(false, false, []string{"LogicalPort", "LogicalSwitch", "NSGroup"}, "List of object where rule will be enforced. The section level field overrides this one. Null will be treated as any"),
				"destination": getResourceReferencesSchema(false, false, []string{"IPSet", "LogicalPort", "LogicalSwitch", "NSGroup", "MACSet"}, "List of the destinations. Null will be treated as any"),
				"destinations_excluded": &schema.Schema{
					Type:        schema.TypeBool,
					Description: "Negation of the destination",
					Optional:    true,
				},
				"direction": &schema.Schema{
					Type:         schema.TypeString,
					Description:  "Rule direction in case of stateless firewall rules. This will only considered if section level parameter is set to stateless. Default to IN_OUT if not specified",
					Optional:     true,
					ValidateFunc: validation.StringInSlice(firewallRuleDirectionValues, false),
				},
				"disabled": &schema.Schema{
					Type:        schema.TypeBool,
					Description: "Flag to disable rule. Disabled will only be persisted but never provisioned/realized",
					Optional:    true,
				},
				"ip_protocol": &schema.Schema{
					Type:         schema.TypeString,
					Description:  "Type of IP packet that should be matched while enforcing the rule (IPV4, IPV6, IPV4_IPV6)",
					Optional:     true,
					ValidateFunc: validation.StringInSlice(firewallRuleIPProtocolValues, false),
				},
				"logged": &schema.Schema{
					Type:        schema.TypeBool,
					Description: "Flag to enable packet logging. Default is disabled",
					Optional:    true,
				},
				"notes": &schema.Schema{
					Type:        schema.TypeString,
					Description: "User notes specific to the rule",
					Optional:    true,
				},
				"rule_tag": &schema.Schema{
					Type:        schema.TypeString,
					Description: "User level field which will be printed in CLI and packet logs",
					Optional:    true,
				},
				"source": getResourceReferencesSchema(false, false, []string{"IPSet", "LogicalPort", "LogicalSwitch", "NSGroup", "MACSet"}, "List of sources. Null will be treated as any"),
				"sources_excluded": &schema.Schema{
					Type:        schema.TypeBool,
					Description: "Negation of the source",
					Optional:    true,
				},
				"service": getResourceReferencesSchema(false, false, []string{"NSService", "NSServiceGroup"}, "List of the services. Null will be treated as any"),
			},
		},
	}
}

func returnServicesResourceReferences(services []manager.FirewallService) []map[string]interface{} {
	var servicesList []map[string]interface{}
	for _, srv := range services {
		elem := make(map[string]interface{})
		elem["is_valid"] = srv.IsValid
		elem["target_display_name"] = srv.TargetDisplayName
		elem["target_id"] = srv.TargetId
		elem["target_type"] = srv.TargetType
		servicesList = append(servicesList, elem)
	}
	return servicesList
}

func setRulesInSchema(d *schema.ResourceData, rules []manager.FirewallRule) {
	var rulesList []map[string]interface{}
	for _, rule := range rules {
		elem := make(map[string]interface{})
		elem["id"] = rule.Id
		elem["display_name"] = rule.DisplayName
		elem["description"] = rule.Description
		elem["rule_tag"] = rule.RuleTag
		elem["notes"] = rule.Notes
		elem["logged"] = rule.Logged
		elem["action"] = rule.Action
		elem["destinations_excluded"] = rule.DestinationsExcluded
		elem["sources_excluded"] = rule.SourcesExcluded
		elem["ip_protocol"] = rule.IpProtocol
		elem["disabled"] = rule.Disabled
		elem["revision"] = rule.Revision
		elem["direction"] = rule.Direction
		elem["source"] = returnResourceReferences(rule.Sources)
		elem["destination"] = returnResourceReferences(rule.Destinations)
		elem["service"] = returnServicesResourceReferences(rule.Services)

		rulesList = append(rulesList, elem)
	}
	d.Set("rule", rulesList)
}

func getServicesResourceReferences(services []interface{}) []manager.FirewallService {
	var servicesList []manager.FirewallService
	for _, srv := range services {
		data := srv.(map[string]interface{})
		elem := manager.FirewallService{
			IsValid:           data["is_valid"].(bool),
			TargetDisplayName: data["target_display_name"].(string),
			TargetId:          data["target_id"].(string),
			TargetType:        data["target_type"].(string),
		}
		servicesList = append(servicesList, elem)
	}
	return servicesList
}

func getRulesFromSchema(d *schema.ResourceData) []manager.FirewallRule {
	rules := d.Get("rule").([]interface{})
	var ruleList []manager.FirewallRule
	for _, rule := range rules {
		data := rule.(map[string]interface{})
		elem := manager.FirewallRule{
			DisplayName:          data["display_name"].(string),
			RuleTag:              data["rule_tag"].(string),
			Notes:                data["notes"].(string),
			Description:          data["description"].(string),
			Action:               data["action"].(string),
			Logged:               data["logged"].(bool),
			Disabled:             data["disabled"].(bool),
			Revision:             int64(data["revision"].(int)),
			SourcesExcluded:      data["sources_excluded"].(bool),
			DestinationsExcluded: data["destinations_excluded"].(bool),
			IpProtocol:           data["ip_protocol"].(string),
			Direction:            data["direction"].(string),
			Sources:              getResourceReferences(data["source"].([]interface{})),
			Destinations:         getResourceReferences(data["destination"].([]interface{})),
			Services:             getServicesResourceReferences(data["service"].([]interface{})),
		}

		ruleList = append(ruleList, elem)
	}
	return ruleList
}

func resourceNsxtFirewallSectionCreateEmpty(d *schema.ResourceData, m interface{}) error {
	nsxClient := m.(*api.APIClient)
	description := d.Get("description").(string)
	displayName := d.Get("display_name").(string)
	tags := getTagsFromSchema(d)
	appliedTos := getResourceReferencesFromSchemaSet(d, "applied_to")
	isDefault := d.Get("is_default").(bool)
	sectionType := d.Get("section_type").(string)
	stateful := d.Get("stateful").(bool)

	localVarOptionals := make(map[string]interface{})
	firewallSection := manager.FirewallSection{
		Description: description,
		DisplayName: displayName,
		Tags:        tags,
		AppliedTos:  appliedTos,
		IsDefault:   isDefault,
		SectionType: sectionType,
		Stateful:    stateful,
	}
	firewallSection, resp, err := nsxClient.ServicesApi.AddSection(nsxClient.Context, firewallSection, localVarOptionals)
	if err != nil {
		return fmt.Errorf("Error during FirewallSection create empty: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Unexpected status returned during FirewallSection create empty: %v", resp.StatusCode)
	}
	d.SetId(firewallSection.Id)

	return resourceNsxtFirewallSectionRead(d, m)
}

func resourceNsxtFirewallSectionCreate(d *schema.ResourceData, m interface{}) error {
	rules := getRulesFromSchema(d)
	if len(rules) == 0 {
		return resourceNsxtFirewallSectionCreateEmpty(d, m)
	}
	nsxClient := m.(*api.APIClient)
	description := d.Get("description").(string)
	displayName := d.Get("display_name").(string)
	tags := getTagsFromSchema(d)
	appliedTos := getResourceReferencesFromSchemaSet(d, "applied_to")
	isDefault := d.Get("is_default").(bool)
	sectionType := d.Get("section_type").(string)
	stateful := d.Get("stateful").(bool)

	firewallSection := manager.FirewallSectionRuleList{
		Description: description,
		DisplayName: displayName,
		Tags:        tags,
		AppliedTos:  appliedTos,
		IsDefault:   isDefault,
		SectionType: sectionType,
		Stateful:    stateful,
		Rules:       rules,
	}
	localVarOptionals := make(map[string]interface{})
	firewallSection, resp, err := nsxClient.ServicesApi.AddSectionWithRulesCreateWithRules(nsxClient.Context, firewallSection, localVarOptionals)
	if err != nil {
		return fmt.Errorf("Error during FirewallSection create with rules: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Unexpected status returned during FirewallSection create with rules: %v", resp.StatusCode)
	}
	d.SetId(firewallSection.Id)

	return resourceNsxtFirewallSectionRead(d, m)
}

func resourceNsxtFirewallSectionRead(d *schema.ResourceData, m interface{}) error {
	nsxClient := m.(*api.APIClient)
	id := d.Id()
	if id == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	firewallSection, resp, err := nsxClient.ServicesApi.GetSectionWithRulesListWithRules(nsxClient.Context, id)
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[DEBUG] FirewallSection %s not found", id)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error during FirewallSection %s read: %v", id, err)
	}

	d.Set("revision", firewallSection.Revision)
	d.Set("description", firewallSection.Description)
	d.Set("display_name", firewallSection.DisplayName)
	setTagsInSchema(d, firewallSection.Tags)
	setRulesInSchema(d, firewallSection.Rules)
	d.Set("is_default", firewallSection.IsDefault)
	d.Set("section_type", firewallSection.SectionType)
	d.Set("stateful", firewallSection.Stateful)

	// Getting the applied tos will require another api call (for NSX 2.1 or less)
	firewallSection2, resp, err := nsxClient.ServicesApi.GetSection(nsxClient.Context, id)
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[DEBUG] FirewallSection %s not found", id)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error during FirewallSection %s read: %v", id, err)
	}
	setResourceReferencesInSchema(d, firewallSection2.AppliedTos, "applied_to")

	return nil
}

func resourceNsxtFirewallSectionUpdateEmpty(d *schema.ResourceData, m interface{}, id string) error {
	nsxClient := m.(*api.APIClient)
	revision := int64(d.Get("revision").(int))
	description := d.Get("description").(string)
	displayName := d.Get("display_name").(string)
	tags := getTagsFromSchema(d)
	appliedTos := getResourceReferencesFromSchemaSet(d, "applied_to")
	isDefault := d.Get("is_default").(bool)
	sectionType := d.Get("section_type").(string)
	stateful := d.Get("stateful").(bool)
	firewallSection := manager.FirewallSection{
		Revision:    revision,
		Description: description,
		DisplayName: displayName,
		Tags:        tags,
		AppliedTos:  appliedTos,
		IsDefault:   isDefault,
		SectionType: sectionType,
		Stateful:    stateful,
	}
	// Update the section ignoring the rules
	firewallSection, resp, err := nsxClient.ServicesApi.UpdateSection(nsxClient.Context, id, firewallSection)

	if err != nil || resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Error during FirewallSection %s update empty: %v", id, err)
	}

	// Read the section, and delete all current rules from it
	currSection, resp2, err2 := nsxClient.ServicesApi.GetSectionWithRulesListWithRules(nsxClient.Context, id)
	if resp2.StatusCode == http.StatusNotFound {
		return fmt.Errorf("FirewallSection %s not found during update empty action", id)
	}
	if err2 != nil {
		return fmt.Errorf("Error during FirewallSection %s update empty: cannot read the section: %v", id, err2)
	}
	for _, rule := range currSection.Rules {
		nsxClient.ServicesApi.DeleteRule(nsxClient.Context, id, rule.Id)
	}
	return resourceNsxtFirewallSectionRead(d, m)
}

func resourceNsxtFirewallSectionUpdate(d *schema.ResourceData, m interface{}) error {
	id := d.Id()
	if id == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	rules := getRulesFromSchema(d)
	if len(rules) == 0 {
		return resourceNsxtFirewallSectionUpdateEmpty(d, m, id)
	}

	nsxClient := m.(*api.APIClient)
	revision := int64(d.Get("revision").(int))
	description := d.Get("description").(string)
	displayName := d.Get("display_name").(string)
	tags := getTagsFromSchema(d)
	appliedTos := getResourceReferencesFromSchemaSet(d, "applied_to")
	isDefault := d.Get("is_default").(bool)
	sectionType := d.Get("section_type").(string)
	stateful := d.Get("stateful").(bool)
	firewallSection := manager.FirewallSectionRuleList{
		Revision:    revision,
		Description: description,
		DisplayName: displayName,
		Tags:        tags,
		AppliedTos:  appliedTos,
		IsDefault:   isDefault,
		SectionType: sectionType,
		Stateful:    stateful,
		Rules:       rules,
	}

	firewallSection, resp, err := nsxClient.ServicesApi.UpdateSectionWithRulesUpdateWithRules(nsxClient.Context, id, firewallSection)

	if err != nil || resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("Error during FirewallSection %s update: %v", id, err)
	}

	return resourceNsxtFirewallSectionRead(d, m)
}

func resourceNsxtFirewallSectionDelete(d *schema.ResourceData, m interface{}) error {
	nsxClient := m.(*api.APIClient)
	id := d.Id()
	if id == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	localVarOptionals := make(map[string]interface{})
	localVarOptionals["cascade"] = true
	resp, err := nsxClient.ServicesApi.DeleteSection(nsxClient.Context, id, localVarOptionals)
	if err != nil {
		return fmt.Errorf("Error during FirewallSection %s delete: %v", id, err)
	}

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("[DEBUG] FirewallSection %s not found", id)
		d.SetId("")
	}
	return nil
}