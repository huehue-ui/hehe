package models

// Campaign represents the structure of a marketing campaign.
type Campaign struct {
	CampaignID     string // Unique identifier for the campaign
	CampaignName   string // Name of the campaign
	ImageURL       string // URL of the campaign image
	CallToAction   string // Call to action text (e.g., "Learn More")
	CampaignStatus string // Status of the campaign (e.g., "active", "inactive")
	CDate          string // Creation date of the campaign
	UDate          string // Last update date of the campaign
}

// TargetingRule defines the rules for targeting a campaign to a specific audience.
type TargetingRule struct {
	CampaignID string // Foreign key linking to the Campaign
	Dimension  string // The dimension to target (e.g., "country", "os", "app_version")
	Type       string // Type of targeting (e.g., "include", "exclude", "equals")
	Value      string // The value for the dimension (e.g., "US", "android", "1.2.0")
	CDate      string // Creation date of the rule
	UDate      string // Last update date of the rule
}

// DeliveryRequest represents the expected structure for a campaign delivery request.
// Note: Currently, the DeliveryHandler parses query parameters directly. This struct might be for future use or an alternative request format.
type DeliveryRequest struct {
	AppID   string `json:"app_id"`   // ID of the application requesting the campaign
	Country string `json:"country"`  // Country code for targeting
	OS      string `json:"os"`       // Operating system for targeting
}

// DeliveryResponse defines the structure of the campaign data sent back to the client.
type DeliveryResponse struct {
	CID string `json:"cid"` // Campaign ID
	Img string `json:"img"` // Image URL for the campaign
	CTA string `json:"cta"` // Call to action text
}
