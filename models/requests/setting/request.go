package setting

import "backend/models/domains"

// UpdateBySubgroupRequest represents request to update settings by subgroup
type UpdateBySubgroupRequest struct {
	Settings []UpdateSettingItem `json:"settings" validate:"required,min=1,dive"`
}

// UpdateSettingItem represents a single setting to update
type UpdateSettingItem struct {
	Name  string `json:"name" validate:"required"`
	Value string `json:"value" validate:"required"`
}

// UpdateAIPromptRequest represents request body for updating an AI prompt section
type UpdateAIPromptRequest struct {
	// The prompt text for this section. Can be empty to clear the prompt.
	Prompt string `json:"prompt" example:"We are a restaurant open Sunday to Friday, serving authentic local cuisine."`
}

// AIPromptSectionResponse represents a single AI prompt section
type AIPromptSectionResponse struct {
	// Section name (product | delivery | operational | about-store | faq)
	Section string `json:"section" example:"about-store"`
	// The prompt text configured for this section
	Prompt string `json:"prompt" example:"We are a restaurant open Sunday to Friday."`
}

// AIPromptsResponse represents all 5 AI prompt sections
type AIPromptsResponse struct {
	Product     string `json:"product"     example:"Show product name, price, photo, and description."`
	Delivery    string `json:"delivery"    example:"We deliver to the following zones."`
	Operational string `json:"operational" example:"Open Sunday - Friday, 07:00 - 17:00."`
	AboutStore  string `json:"about_store" example:"We are a restaurant located in Jakarta."`
	Faq         string `json:"faq"         example:"1. Payment via Stripe. 2. Order expires in 15 minutes."`
}

// ToSettings converts request to domain settings
func (r *UpdateBySubgroupRequest) ToSettings(subGroupName string) []domains.Setting {
	settings := make([]domains.Setting, len(r.Settings))
	for i, s := range r.Settings {
		settings[i] = domains.Setting{
			SubgroupName: subGroupName,
			Name:         s.Name,
			Value:        s.Value,
		}
	}
	return settings
}
