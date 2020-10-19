package kit

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/client/alertgroup"
	"github.com/prometheus/alertmanager/api/v2/client/general"
	"github.com/prometheus/alertmanager/api/v2/client/receiver"
	"github.com/prometheus/alertmanager/api/v2/client/silence"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/cli"
)

type AlertmanagerClient struct {
	config struct {
		scheme, host     string
		port, targetPort *int
	}
	balancer *client.Alertmanager
	backends map[string]*client.Alertmanager
	mutex    sync.Mutex
}

func NewClient(config ClientConfig) (*AlertmanagerClient, error) {
	c := &AlertmanagerClient{backends: make(map[string]*client.Alertmanager)}
	c.config.scheme = "http"

	if config.URL != "" {
		su, e := url.Parse(config.URL)
		if e != nil {
			return nil, e
		}
		hostport := strings.Split(su.Host, ":")
		c.config.host = hostport[0]
		if len(hostport) > 1 {
			i, e := strconv.Atoi(hostport[1])
			if e == nil {
				c.config.port = &i
			} else {
				return nil, fmt.Errorf("error converting port：%v", e)
			}
		}
		// Additional targetPort can be configured when using url configuration.
		if config.Service != nil {
			c.config.targetPort = config.Service.TargetPort
		}
	} else if svc := config.Service; svc != nil {
		c.config.host = fmt.Sprintf("%s.%s.svc", svc.Name, svc.Namespace)
		c.config.port = svc.Port
		c.config.targetPort = svc.TargetPort
	} else {
		c.config.host = "localhost"
	}
	if c.config.port == nil {
		defPort := 9093
		c.config.port = &defPort
	}
	if c.config.targetPort == nil {
		c.config.targetPort = c.config.port
	}
	u, e := url.Parse(fmt.Sprintf("%s://%s:%d",
		c.config.scheme, c.config.host, *c.config.port))
	if e != nil {
		return nil, e
	}
	c.balancer = cli.NewAlertmanagerClient(u)
	return c, nil
}

// GetAlerts gets alerts by the filter.
func (c *AlertmanagerClient) GetAlerts(ctx context.Context, af *AlertsFilter) ([]*Alert, error) {
	p := toGetAlertsParams(ctx, af)
	as, e := c.balancer.Alert.GetAlerts(p)
	if e != nil {
		return nil, e
	}
	return fromGettableAlerts(as.Payload), nil
}

// GetAlerts gets alert groups by the filter.
func (c *AlertmanagerClient) GetAlertGroups(ctx context.Context, af *AlertsFilter) ([]*AlertGroup, error) {
	p := toGetAlertGroupsParams(ctx, af)
	ags, e := c.balancer.Alertgroup.GetAlertGroups(p)
	if e != nil {
		return nil, e
	}
	return fromAlertGroups(ags.Payload), nil
}

// PostAlerts posts alerts to alertmanager.
// Alerts will be posted to every instance behind alertmanager service.
func (c *AlertmanagerClient) PostAlerts(ctx context.Context, alerts []*RawAlert) error {
	p := toPostAlertsParams(ctx, alerts)
	var ams []*client.Alertmanager
	astatus, e := c.GetStatus(ctx)
	if e != nil {
		return e
	}
	peerHosts := c.peerHosts(astatus)
	if len(peerHosts) <= 1 { // use balancer if only one instance behind the service
		ams = append(ams, c.balancer)
	} else {
		bks, e := c.getBackends(peerHosts)
		if e != nil {
			return e
		}
		ams = append(ams, bks...)
	}
	for _, am := range ams {
		if _, e := am.Alert.PostAlerts(p); e != nil {
			return e
		}
	}
	return nil
}

// GetSilence gets silence by silence id.
func (c *AlertmanagerClient) GetSilence(ctx context.Context, silenceId string) (*Silence, error) {
	p := silence.NewGetSilenceParamsWithContext(ctx).
		WithSilenceID(strfmt.UUID(silenceId))
	gs, e := c.balancer.Silence.GetSilence(p)
	if e != nil {
		return nil, e
	}
	return fromSilence(gs.Payload), nil
}

// GetSilences gets silences by filter
// filter supports a simplified prometheus query syntax, contains operators: =, !=, =~, !~ .
// This will be used to filter your query by value matching or regex matching, and their negative.
func (c *AlertmanagerClient) GetSilences(ctx context.Context, filter []string) ([]*Silence, error) {
	p := silence.NewGetSilencesParamsWithContext(ctx).
		WithFilter(filter)
	gss, e := c.balancer.Silence.GetSilences(p)
	if e != nil {
		return nil, e
	}
	var ss []*Silence
	for _, gs := range gss.Payload {
		ss = append(ss, fromSilence(gs))
	}
	return ss, nil
}

func (c *AlertmanagerClient) PostSilence(ctx context.Context, rsil *RawSilence) (string, error) {
	p := toPostSilenceParams(ctx, rsil)
	ps, e := c.balancer.Silence.PostSilences(p)
	if e != nil {
		return "", e
	}
	return ps.Payload.SilenceID, nil
}

func (c *AlertmanagerClient) DeleteSilence(ctx context.Context, silId string) error {
	p := silence.NewDeleteSilenceParamsWithContext(ctx).
		WithSilenceID(strfmt.UUID(silId))
	_, e := c.balancer.Silence.DeleteSilence(p)
	if e != nil {
		return e
	}
	return nil
}

func (c *AlertmanagerClient) GetReceivers(ctx context.Context) ([]*Receiver, error) {
	p := receiver.NewGetReceiversParamsWithContext(ctx)
	rs, e := c.balancer.Receiver.GetReceivers(p)
	if e != nil {
		return nil, e
	}
	var krs []*Receiver
	for _, r := range rs.Payload {
		krs = append(krs, &Receiver{
			Name: *r.Name,
		})
	}
	return krs, nil
}

func (c *AlertmanagerClient) GetStatus(ctx context.Context) (*AlertmanagerStatus, error) {
	s, e := c.balancer.General.GetStatus(&general.GetStatusParams{Context: ctx})
	if e != nil {
		return nil, e
	}
	return (*AlertmanagerStatus)(s.Payload), nil
}

func (c *AlertmanagerClient) peerHosts(s *AlertmanagerStatus) []string {
	if s.Cluster == nil {
		return nil
	}
	var phosts []string
	for _, ps := range s.Cluster.Peers {
		if phost := strings.Split(*ps.Address, ":")[0]; phost != "" {
			phosts = append(phosts, phost)
		}
	}
	return phosts
}

func (c *AlertmanagerClient) getBackends(peerHosts []string) ([]*client.Alertmanager, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	phmap := make(map[string]struct{})
	for _, ph := range peerHosts {
		u := fmt.Sprintf("%s://%s:%d", c.config.scheme, ph, *c.config.targetPort)
		phmap[u] = struct{}{}
		if _, ok := c.backends[u]; !ok {
			uu, e := url.Parse(u)
			if e != nil {
				return nil, e
			}
			c.backends[u] = cli.NewAlertmanagerClient(uu)
		}
	}
	for u, _ := range c.backends {
		if _, ok := phmap[u]; !ok {
			delete(c.backends, u)
		}
	}

	var ams []*client.Alertmanager
	for _, am := range c.backends {
		ams = append(ams, am)
	}
	return ams, nil
}

func fromGettableAlerts(alerts models.GettableAlerts) []*Alert {
	var kas []*Alert
	for _, a := range alerts {
		var rs []*Receiver
		for _, r := range a.Receivers {
			rs = append(rs, &Receiver{
				Name: *r.Name,
			})
		}
		ka := &Alert{
			RawAlert: RawAlert{
				Labels:      a.Labels,
				Annotations: a.Annotations,
				StartsAt:    *a.StartsAt,
				EndsAt:      *a.EndsAt,
			},
			Fingerprint: *a.Fingerprint,
			Receivers:   rs,
		}
		if s := a.Status; s != nil {
			ka.Status = &AlertStatus{
				InhibitedBy: s.InhibitedBy,
				SilencedBy:  s.SilencedBy,
				State:       *s.State,
			}
		}
		kas = append(kas, ka)
	}
	return kas
}

func fromAlertGroups(ags models.AlertGroups) []*AlertGroup {
	var kags []*AlertGroup
	for _, ag := range ags {
		kag := &AlertGroup{
			Alerts: fromGettableAlerts(ag.Alerts),
			Labels: ag.Labels,
		}
		if r := ag.Receiver; r != nil {
			kag.Receiver = &Receiver{Name: *r.Name}
		}
		kags = append(kags, kag)
	}
	return kags
}

func toGetAlertsParams(ctx context.Context, af *AlertsFilter) *alert.GetAlertsParams {
	return alert.NewGetAlertsParams().
		WithContext(ctx).
		WithActive(&af.Active).
		WithInhibited(&af.Inhibited).
		WithSilenced(&af.Silenced).
		WithUnprocessed(&af.Unprocessed).
		WithFilter(af.Filter).
		WithReceiver(&af.Receiver)
}

func toGetAlertGroupsParams(ctx context.Context, af *AlertsFilter) *alertgroup.GetAlertGroupsParams {
	return alertgroup.NewGetAlertGroupsParams().
		WithContext(ctx).
		WithActive(&af.Active).
		WithInhibited(&af.Inhibited).
		WithSilenced(&af.Silenced).
		WithFilter(af.Filter).
		WithReceiver(&af.Receiver)
}

func toPostAlertsParams(ctx context.Context, alerts []*RawAlert) *alert.PostAlertsParams {
	var as models.PostableAlerts
	for _, a := range alerts {
		as = append(as, &models.PostableAlert{
			Alert:       models.Alert{Labels: a.Labels},
			Annotations: a.Annotations,
			StartsAt:    a.StartsAt,
			EndsAt:      a.EndsAt,
		})
	}
	return alert.NewPostAlertsParams().
		WithContext(ctx).
		WithAlerts(as)
}

func fromSilence(gs *models.GettableSilence) *Silence {
	if gs == nil {
		return nil
	}
	var ms []*Matcher
	for _, m := range gs.Matchers {
		ms = append(ms, &Matcher{
			IsRegex: *m.IsRegex,
			Name:    *m.Name,
			Value:   *m.Value,
		})
	}
	return &Silence{
		RawSilence: RawSilence{
			ID:        *gs.ID,
			StartsAt:  *gs.StartsAt,
			EndsAt:    *gs.EndsAt,
			Comment:   *gs.Comment,
			CreatedBy: *gs.CreatedBy,
			Matchers:  ms,
		},
		Status: &SilenceStatus{
			State: *gs.Status.State,
		},
		UpdatedAt: *gs.UpdatedAt,
	}
}

func toPostSilenceParams(ctx context.Context, rsil *RawSilence) *silence.PostSilencesParams {
	var ms []*models.Matcher
	for _, m := range rsil.Matchers {
		ms = append(ms, &models.Matcher{
			IsRegex: &m.IsRegex,
			Name:    &m.Name,
			Value:   &m.Value,
		})
	}
	return silence.NewPostSilencesParamsWithContext(ctx).
		WithSilence(&models.PostableSilence{
			ID: rsil.ID,
			Silence: models.Silence{
				StartsAt:  &rsil.StartsAt,
				EndsAt:    &rsil.EndsAt,
				Comment:   &rsil.Comment,
				CreatedBy: &rsil.CreatedBy,
				Matchers:  ms,
			},
		})
}
