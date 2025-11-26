package models

// ProcessRequest representa el request para procesar órdenes
type ProcessRequest struct {
	APIKey             string `json:"api_key"`
	Date               string `json:"date"`
	DropiCountrySuffix string `json:"dropi_country_suffix"`
	WebhookSuffix      string `json:"webhook_suffix"`
	DateUtil           string `json:"date_util,omitempty"`

	IDWorkspace string `json:"id_workspace,omitempty"`
	FlowNS      string `json:"flow_ns,omitempty"`
}

// GetDropiCountrySuffix implementa la interfaz del validator
func (p *ProcessRequest) GetDropiCountrySuffix() string {
	return p.DropiCountrySuffix
}

// GetWebhookSuffix implementa la interfaz del validator
func (p *ProcessRequest) GetWebhookSuffix() string {
	return p.WebhookSuffix
}
