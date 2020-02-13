package pdaclient

import (
	"time"
	"encoding/json"

	"github.com/PagerDuty/go-pagerduty"
)


type PdaEvent struct {
	Id            string                  `json:"id"`
	Agent         Agent                   `json:"agent"`
	Event         pagerduty.Event         `json:"event"`
	EventResponse pagerduty.EventResponse `json:"event_response"`
}

type Agent struct {
	AgentId  string    `json:"agent_id"`
	QueuedAt time.Time `json:"queued_at"`
	QueuedBy string    `json:"queued_by"`
	Retry    int       `json:"retry"`
}

func QueueEvent(eventType string,
	serviceKey string,
	incidentKey string,
	description string,
	client string,
	clientUrl string,
	details map[string]string,
	agentId string,
	queuedBy string) (PdaEvent, error) {
	var pda PdaEvent
	var a Agent
	a.AgentId     = agentId
	a.QueuedBy    = queuedBy
	a.QueuedAt    = time.Now()

	var e pagerduty.Event
	e.ServiceKey  = serviceKey
	e.Type        = eventType
	e.IncidentKey = incidentKey
	e.Description = description
	e.Client      = client
	e.ClientURL   = clientUrl
	e.Details     = details
	pda.Agent       = a
	pda.Event       = e

	return pda, nil
}

func (e PdaEvent) EventToBytes() ([]byte) {
	jsonBytes, err := json.Marshal(e)
	if err != nil {
		return nil
	}
	return jsonBytes
}
