package kit

import (
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/models"
)

type RawAlert struct {
	Labels models.LabelSet `json:"labels,omitempty"`
	Annotations models.LabelSet `json:"annotations,omitempty"`
	StartsAt strfmt.DateTime `json:"startsAt,omitempty"`
	EndsAt strfmt.DateTime `json:"endsAt,omitempty"`
}

type Alert struct {
	RawAlert
	Fingerprint string `json:"fingerprint"`
	Receivers []*Receiver `json:"receivers"`
	Status *AlertStatus `json:"status"`
}

type AlertStatus struct {
	InhibitedBy []string `json:"inhibitedBy"`
	SilencedBy []string `json:"silencedBy"`
	State string `json:"state"`
}

type Receiver struct {
	Name string `json:"name"`
}

type AlertGroup struct {
	Alerts []*Alert `json:"alerts"`
	Labels models.LabelSet `json:"labels"`
	Receiver *Receiver `json:"receiver"`
}

type AlertsFilter struct {
	Active bool `json:"active"`
	Inhibited bool `json:"inhibited"`
	Silenced bool `json:"silenced"`
	Unprocessed bool `json:"unprocessed"`
	// filter supports a simplified prometheus query syntax, contains operators: =, !=, =~, !~ .
	// This will be used to filter your query by value matching or regex matching, and their negative.
	Filter []string `json:"filter"`
	Receiver string `json:"receiver"`
}

func (f *AlertsFilter) WithActive(active bool) *AlertsFilter {
	f.Active = active
	return f
}
func (f *AlertsFilter) WithInhibited(inhibited bool) *AlertsFilter {
	f.Inhibited = inhibited
	return f
}
func (f *AlertsFilter) WithSilenced(silenced bool) *AlertsFilter {
	f.Silenced = silenced
	return f
}
func (f *AlertsFilter) WithUnprocessed(unprocessed bool) *AlertsFilter {
	f.Unprocessed = unprocessed
	return f
}
func (f *AlertsFilter) WithFilter(filter []string) *AlertsFilter {
	f.Filter = filter
	return f
}
func (f *AlertsFilter) WithReceiver(receiver string) *AlertsFilter {
	f.Receiver = receiver
	return f
}

func NewAlertsFilter() *AlertsFilter {
	return &AlertsFilter{
		Active: true,
		Inhibited: true,
		Silenced: true,
		Unprocessed: true,
	}
}

type RawSilence struct {
	ID string `json:"id"`
	StartsAt strfmt.DateTime `json:"startsAt,omitempty"`
	EndsAt strfmt.DateTime `json:"endsAt,omitempty"`
	Comment string `json:"comment"`
	CreatedBy string `json:"createdBy"`
	Matchers []*Matcher `json:"matchers"`
}

type Silence struct {
	RawSilence
	Status *SilenceStatus `json:"status"`
	UpdatedAt strfmt.DateTime `json:"updatedAt"`
}

type SilenceStatus struct {
	State string `json:"state"`
}

type Matcher struct {
	IsRegex bool `json:"isRegex"`
	Name string `json:"name"`
	Value string `json:"value"`
}

type AlertmanagerStatus models.AlertmanagerStatus