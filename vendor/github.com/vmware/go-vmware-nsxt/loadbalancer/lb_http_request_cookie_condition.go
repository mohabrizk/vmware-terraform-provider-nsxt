/*
 * NSX API
 *
 * VMware NSX REST API
 *
 * API version: 1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package loadbalancer

// This condition is used to match HTTP request messages by cookie which is a specific type of HTTP header. The match_type and case_sensitive define how to compare cookie value.
type LbHttpRequestCookieCondition struct {

	// A flag to indicate whether reverse the match result of this condition
	Inverse bool `json:"inverse,omitempty"`

	// Type of load balancer rule condition
	Type_ string `json:"type"`

	// If true, case is significant when comparing cookie value.
	CaseSensitive bool `json:"case_sensitive,omitempty"`

	// Name of cookie
	CookieName string `json:"cookie_name"`

	// Value of cookie
	CookieValue string `json:"cookie_value"`

	// Match type of cookie value
	MatchType string `json:"match_type,omitempty"`
}
