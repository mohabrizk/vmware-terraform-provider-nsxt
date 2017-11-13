package nsxt

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	api "github.com/vmware/go-vmware-nsxt"
	"github.com/vmware/go-vmware-nsxt/manager"
	"net/http"
)

func resourceL4PortSetNsService() *schema.Resource {
	return &schema.Resource{
		Create: resourceL4PortSetNsServiceCreate,
		Read:   resourceL4PortSetNsServiceRead,
		Update: resourceL4PortSetNsServiceUpdate,
		Delete: resourceL4PortSetNsServiceDelete,

		Schema: map[string]*schema.Schema{
			"revision": GetRevisionSchema(),
			"system_owned": GetSystemOwnedSchema(),
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
			"tags": GetTagsSchema(),
			"default_service": &schema.Schema{
				Type:        schema.TypeBool,
				Description: "The default NSServices are created in the system by default. These NSServices can't be modified/deleted",
				Computed:    true,
			},
			"destination_ports": &schema.Schema{
				Type:        schema.TypeSet,
				Description: "Set of destination ports",
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"source_ports": &schema.Schema{
				Type:        schema.TypeSet,
				Description: "Set of source ports",
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
			},
			"l4_protocol": &schema.Schema{
				Type:        schema.TypeString,
				Description: "L4 Protocol",
				Required:    true,
			},
		},
	}
}

func resourceL4PortSetNsServiceCreate(d *schema.ResourceData, m interface{}) error {

	nsxClient := m.(*api.APIClient)

	description := d.Get("description").(string)
	display_name := d.Get("display_name").(string)
	tags := GetTagsFromSchema(d)
	default_service := d.Get("default_service").(bool)
	l4_protocol := d.Get("l4_protocol").(string)
	source_ports := Interface2StringList(d.Get("source_ports").(*schema.Set).List())
	destination_ports := Interface2StringList(d.Get("destination_ports").(*schema.Set).List())

	ns_service := manager.L4PortSetNsService{
		NsService: manager.NsService{
			Description:    description,
			DisplayName:    display_name,
			Tags:           tags,
			DefaultService: default_service,
		},
		NsserviceElement: manager.L4PortSetNsServiceEntry{
			ResourceType:     "L4PortSetNSService",
			L4Protocol:       l4_protocol,
			DestinationPorts: destination_ports,
			SourcePorts:      source_ports,
		},
	}

	ns_service, resp, err := nsxClient.GroupingObjectsApi.CreateL4PortSetNSService(nsxClient.Context, ns_service)

	if err != nil {
		return fmt.Errorf("Error during NsService create: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Unexpected status returned")
		return nil
	}
	d.SetId(ns_service.Id)
	return resourceL4PortSetNsServiceRead(d, m)
}

func resourceL4PortSetNsServiceRead(d *schema.ResourceData, m interface{}) error {

	nsxClient := m.(*api.APIClient)

	id := d.Id()
	if id == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	ns_service, resp, err := nsxClient.GroupingObjectsApi.ReadL4PortSetNSService(nsxClient.Context, id)
	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("NsService not found")
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error during NsService read: %v", err)
	}

	nsservice_element := ns_service.NsserviceElement

	d.Set("revision", ns_service.Revision)
	d.Set("system_owned", ns_service.SystemOwned)
	d.Set("description", ns_service.Description)
	d.Set("display_name", ns_service.DisplayName)
	SetTagsInSchema(d, ns_service.Tags)
	d.Set("default_service", ns_service.DefaultService)
	d.Set("resource_type", nsservice_element.ResourceType)
	d.Set("destination_ports", nsservice_element.DestinationPorts)
	d.Set("source_ports", nsservice_element.SourcePorts)

	return nil
}

func resourceL4PortSetNsServiceUpdate(d *schema.ResourceData, m interface{}) error {

	nsxClient := m.(*api.APIClient)

	id := d.Id()
	if id == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	description := d.Get("description").(string)
	display_name := d.Get("display_name").(string)
	tags := GetTagsFromSchema(d)
	default_service := d.Get("default_service").(bool)
	l4_protocol := d.Get("l4_protocol").(string)
	source_ports := Interface2StringList(d.Get("source_ports").(*schema.Set).List())
	destination_ports := Interface2StringList(d.Get("destination_ports").(*schema.Set).List())
	revision := int64(d.Get("revision").(int))

	ns_service := manager.L4PortSetNsService{
		NsService: manager.NsService{
			Description:    description,
			DisplayName:    display_name,
			Tags:           tags,
			DefaultService: default_service,
			Revision:       revision,
		},
		NsserviceElement: manager.L4PortSetNsServiceEntry{
			ResourceType:     "L4PortSetNSService",
			L4Protocol:       l4_protocol,
			DestinationPorts: destination_ports,
			SourcePorts:      source_ports,
		},
	}

	ns_service, resp, err := nsxClient.GroupingObjectsApi.UpdateL4PortSetNSService(nsxClient.Context, id, ns_service)
	if err != nil || resp.StatusCode == http.StatusNotFound{
		return fmt.Errorf("Error during NsService update: %v %v", err, resp)
	}

	return resourceL4PortSetNsServiceRead(d, m)
}

func resourceL4PortSetNsServiceDelete(d *schema.ResourceData, m interface{}) error {

	nsxClient := m.(*api.APIClient)

	id := d.Id()
	if id == "" {
		return fmt.Errorf("Error obtaining logical object id")
	}

	localVarOptionals := make(map[string]interface{})
	resp, err := nsxClient.GroupingObjectsApi.DeleteNSService(nsxClient.Context, id, localVarOptionals)
	if err != nil {
		return fmt.Errorf("Error during NsService delete: %v", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("NsService not found")
		d.SetId("")
	}
	return nil
}