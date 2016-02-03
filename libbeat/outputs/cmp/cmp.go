package cmp

import (
    "fmt"
	"encoding/json"
	"os"
    "bytes"
    "net/http"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
    "github.com/elastic/beats/libbeat/outputs"
)

func init() {
	outputs.RegisterOutputPlugin("cmp", plugin{})
}

type plugin struct{}

func (p plugin) NewOutput(
	config *outputs.MothershipConfig,
	topologyExpire int,
) (outputs.Outputer, error) {
    url := config.Url
    resource_id := config.ResourceID
    headers := config.Headers
	return newCmp(url, resource_id, headers), nil
}

type cmp struct {
	url         string
	resource_id string
	headers     map[string]string
}

func newCmp(url string, customer_id string, headers map[string]string) *cmp {
	return &cmp{url, customer_id, headers}
}

func writeBuffer(buf []byte) error {
	written := 0
	for written < len(buf) {
		n, err := os.Stdout.Write(buf[written:])
		if err != nil {
			return err
		}

		written += n
	}
	return nil
}

func force_to_mapstr(data interface{}) common.MapStr {

    data_json, _ := json.Marshal(data)

    var new_load common.MapStr
    json.Unmarshal(data_json, &new_load)

    return new_load
}

// This is a dump of the event data structure for reference
// {"@timestamp":"2016-02-02T16:13:18.571Z",
//  "beat":{"hostname":"bespin","name":"bespin"},
//   "count":1,
//    "cpu":{"idle":34638722,"iowait":38858,"irq":44,"nice":377,"softirq":76688,"steal":0,"system":469312,"system_p":0,"user":687363,"user_p":0},
//    "load":{"load1":0.74,"load5":0.69,"load15":0.73},
//    "mem":{"actual_free":4368953344,"actual_used":4004057088,"actual_used_p":0.48,"free":1103282176,"total":8373010432,"used":7269728256,"used_p":0.87},
//    "swap":{"free":0,"total":0,"used":0,"used_p":0},
//    "type":"system"}

func postToCmp(c *cmp, event common.MapStr) error {

    url := c.url
    resource_id := c.resource_id
    headers := c.headers

    if event["type"] == "system" {
        cpu := event["cpu"].(common.MapStr)
        load := force_to_mapstr(event["load"])
        mem := force_to_mapstr(event["mem"])
// TODO  swap := event["swap"].(common.MapStr)

        cmp_data := []common.MapStr{}

        cpu_items := []string{"idle", "iowait","irq","nice","softirq","steal","system","user"}
        for _,item := range cpu_items {
            cmp_data = append(cmp_data, common.MapStr{
                "metric": "cpu-usage." + item,
                "unit": "percent",
                "value": cpu[item],
                "resource_id": resource_id,
            })
        }

        load_items := []string{"1", "5","15"}
        for _,item := range load_items {
            cmp_data = append(cmp_data, common.MapStr{
                "metric": "load-avg." + item,
                "unit": "percent",
                "value": load["load" + item],
                "resource_id": resource_id,
            })
        }

        cmp_data = append(cmp_data, common.MapStr{
            "metric": "memory-usage",
            "unit": "percent",
            "value": mem["actual_used_p"].(float64) * 100,
            "resource_id": resource_id,
        })

        // fmt.Printf("cmp_data: %v\n", cmp_data)

        var jsonEvent []byte
        var err error
        jsonEvent, err = json.Marshal(cmp_data)

        req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonEvent))
        req.Header.Set("Content-Type", "application/json")

        for key, value := range headers {
            req.Header.Set(key, value)
        }

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            panic(err)
        }
        defer resp.Body.Close()

        fmt.Println(resp.Status)

    } else {

//        fmt.Printf("EVENT: %v\n", event)
    }

    return nil
}

func (c *cmp) PublishEvent(
	s outputs.Signaler,
	opts outputs.Options,
	event common.MapStr,
) error {
	var err error

	if err = postToCmp(c, event); err != nil {
		goto fail
	}

	outputs.SignalCompleted(s)
	return nil
fail:
	if opts.Guaranteed {
		logp.Critical("Unable to publish events to cmp: %v", err)
	}
	outputs.SignalFailed(s, err)
	return err
}
