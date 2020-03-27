package kit

// ClientConfig contains the information to make a connection with the alertmanger
// If you need to post alerts to alertmanager, and you have customized the target port of alertmanager service,
// please configure ClientConfig.Service.Target, regardless of whether you configure URL.
type ClientConfig struct {
	URL string `json:"url,omitempty"`
	Service *ServiceReference `json:"service,omitemtpy"`
}

type ServiceReference struct {
	Namespace string `json:"namespace"`
	Name string `json:"name"`
	// The port that will be exposed by this service. Defaults to 9093
	Port *int `json:"port,omitempty"`
	// TargetPort is the port to access on the backend instances targeted by the service.
	// If this is not specified, the value of the 'port' field is used.
	TargetPort *int `json:"targetPort,omitempty"`
}