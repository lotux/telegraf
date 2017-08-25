package icinga2

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	client "github.com/lotux/go-icinga2-client/icinga2"


	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/outputs"
)

type Icinga2 struct {
	Prefix string

	URL string

	Username string
	Password string

	Debug bool
	c *client.WebClient
}

type CheckResult struct {
	ExitStatus int `json:"exit_status"`
	PluginOutput string `json:"plugin_output"`
	PerformanceData []string `json:"performance_data"`
	CheckSource string `json:"check_source"`
}

var sampleConfig = `
  ## prefix for metrics keys
  prefix = "my.specific.prefix."

  ## URL for icinga2 API endpoint
  url = "https://icinga2.example.com"

  ## API username / password
  username = user
  password = pass

  ## Debug true - Prints Icinga2 communication
  debug = false
`

func ToLineFormat(tags map[string]string) string {
	tagsArray := make([]string, len(tags))
	index := 0
	for k, v := range tags {
		tagsArray[index] = fmt.Sprintf("%s=%s", k, v)
		index++
	}
	sort.Strings(tagsArray)
	return strings.Join(tagsArray, " ")
}

func (o *Icinga2) Connect() error {
	// Test Connection to Icinga2 Server
	_, err := url.Parse(o.URL)
	if err != nil {
		return fmt.Errorf("Error in parsing host url: %s", err.Error())
	}

	wc, err := client.New(client.WebClient{
		URL:         o.URL,
		Username:    o.Username,
		Password:    o.Password,
		Debug:       false,
		InsecureTLS: true})
	o.c = wc
	return nil
}

func (o *Icinga2) Write(metrics []telegraf.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	u, err := url.Parse(o.URL)
	if err != nil {
		return fmt.Errorf("Error in parsing host url: %s", err.Error())
	}

	if 	u.Scheme == "https" {
		return o.WriteHttp(metrics, u)
	} else {
		return fmt.Errorf("Unknown scheme in host parameter.")
	}
}

func (o *Icinga2) WriteHttp(metrics []telegraf.Metric, u *url.URL) error {

	fmt.Printf("Metrics:%v", metrics)

	for _, m := range metrics {
		host := m.Tags()["host"]
		fmt.Printf("Tags:%v", m.Tags())
		plugin_output := ""
		performance_data := []string{}

		service_name := fmt.Sprintf("%s!%s", host, m.Name())
        _, err := o.c.GetService(service_name)
        if err != nil {
		o.c.CreateService(client.Service{
					Name: m.Name(),
					HostName: host,
					CheckCommand: "hostalive"})
		}
		for fieldName, value := range m.Fields() {
			plugin_output += fmt.Sprintf("%s:%v ", fieldName, value)
			performance_data = append(performance_data,fmt.Sprintf("%s=%v;;;", fieldName, value))
		}
		o.c.ProcessCheckResult(
				host,
				m.Name(),
				client.CheckResult{
				ExitStatus: 0,
				PluginOutput: plugin_output,
				PerformanceData: performance_data,
				CheckSource: host})
	}

	return nil
}

func (o *Icinga2) SampleConfig() string {
	return sampleConfig
}

func (o *Icinga2) Description() string {
	return "Configuration for Icinga2 server to send metrics to"
}

func (o *Icinga2) Close() error {
	return nil
}

func init() {
	outputs.Add("icinga2", func() telegraf.Output {
		return &Icinga2{}
	})
}
