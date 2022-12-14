/*
 * NSX-T Manager API
 *
 * VMware NSX-T Manager REST API
 *
 * API version: 3.2.0.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package supportbundle

type SupportBundleContainerNode struct {
	// Support bundle container type
	ContainerType string `json:"container_type"`
	// List of ContainerClusterNodes identifying container clusters and their nodes
	Clusters []ContainerClusterNode `json:"clusters,omitempty"`
}
