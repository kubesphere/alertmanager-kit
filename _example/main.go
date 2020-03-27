package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-openapi/strfmt"
	"time"

	"github.com/kubesphere/alertmanager-kit"
)

func main() {
	// Init the config
	//config := kit.ClientConfig{
	//	URL: "http://localhost:9093",
	//}
	//Use this if running on kubernetes
	config := kit.ClientConfig{
		Service: &kit.ServiceReference{
			Namespace: "monitoring",
			Name: "alertmanager",
		},
	}

	// Create client for alertmanager service
	client, e := kit.NewClient(config)
	if e != nil {
		panic(e)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)

	// Get status
	status, e := client.GetStatus(ctx)
	prettyPrintlnOrPanic(status, e)

	// Post alerts
	e = client.PostAlerts(ctx, []*kit.RawAlert{{
		Labels:      map[string]string{
			"alertname": "test",
			"alerttype": "test",
			"namespace": "test",
			"pod": "test"},
		Annotations: map[string]string{"message": "test"},
	}})
	prettyPrintlnOrPanic(nil, e)

	// Get alerts
	alerts, e := client.GetAlerts(ctx,
		kit.NewAlertsFilter().WithFilter([]string{"alertname=\"test\""}))
	prettyPrintlnOrPanic(alerts, e)

	// Post silence
	silenceId, e := client.PostSilence(ctx, &kit.RawSilence{
		StartsAt: strfmt.DateTime(time.Now()),
		EndsAt: strfmt.DateTime(time.Now().Add(time.Minute)),
		CreatedBy: "test",
		Matchers: []*kit.Matcher{{
			Name: "alertname",
			Value: "test",
		}},
	})
	prettyPrintlnOrPanic(silenceId, e)

	// Get silence
	silence, e := client.GetSilence(ctx, silenceId)
	prettyPrintlnOrPanic(silence, e)

	// Delete silence
	e = client.DeleteSilence(ctx, silenceId)
	prettyPrintlnOrPanic(nil, e)
}

func prettyPrintlnOrPanic(v interface{}, e error) {
	fmt.Println("----------")
	if e != nil {
		panic(e)
	}
	if v == nil {
		return
	}
	if _, ok := v.(string); ok {
		fmt.Println(v)
		return
	}
	bs, e := json.MarshalIndent(v, "", "\t")
	if e != nil {
		panic(e)
	}
	fmt.Println(string(bs))
}


