package mpsnmp

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	g "github.com/soniah/gosnmp"
	mp "github.com/mackerelio/go-mackerel-plugin-helper"
)

// SNMPMetrics metrics
type SNMPMetrics struct {
	OIDS     []string
	Metrics  mp.Metrics
}

// SNMPPlugin mackerel plugin for snmp
type SNMPPlugin struct {
	GraphName        string
	GraphUnit        string
	Host             string
	Community        string
	Tempfile         string
	SNMPMetricsSlice []SNMPMetrics
}

// FetchMetrics interface for mackerelplugin
func (m SNMPPlugin) FetchMetrics() (map[string]interface{}, error) {
	stat := make(map[string]interface{})

	g.Default.Target = m.Host
	g.Default.Community = m.Community
	err := g.Default.Connect()
	if err != nil {
		log.Println(err)
	}
	defer g.Default.Conn.Close()

	for _, sm := range m.SNMPMetricsSlice {
		resp, err := g.Default.Get(sm.OIDS)
		if err != nil {
			log.Println("SNMP get failed: ", err)
			continue
		}

		ret, err := strconv.ParseFloat(fmt.Sprint(resp.Variables[0].Value), 64)
		if err != nil {
			log.Println(err)
			continue
		}

		stat[sm.Metrics.Name] = ret
	}

	return stat, err
}

// GraphDefinition interface for mackerelplugin
func (m SNMPPlugin) GraphDefinition() map[string]mp.Graphs {
	metrics := []mp.Metrics{}
	for _, sm := range m.SNMPMetricsSlice {
		metrics = append(metrics, sm.Metrics)
	}

	return map[string]mp.Graphs{
		m.GraphName: {
			Label:   m.GraphName,
			Unit:    m.GraphUnit,
			Metrics: metrics,
		},
	}
}

// Do the plugin
func Do() {
	optGraphName := flag.String("name", "snmp", "Graph name")
	optGraphUnit := flag.String("unit", "float", "Graph unit")

	optHost := flag.String("host", "localhost", "Hostname")
	optCommunity := flag.String("community", "public", "SNMP V2c Community")

	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	var snmp SNMPPlugin
	snmp.Host = *optHost
	snmp.Community = *optCommunity
	snmp.GraphName = *optGraphName
	snmp.GraphUnit = *optGraphUnit

	sms := []SNMPMetrics{}
	for _, arg := range flag.Args() {
		vals := strings.Split(arg, ":")
		if len(vals) < 2 {
			continue
		}

		mpm := mp.Metrics{Name: vals[1], Label: vals[1]}
		if len(vals) >= 3 {
			mpm.Diff, _ = strconv.ParseBool(vals[2])
		}
		if len(vals) >= 4 {
			mpm.Stacked, _ = strconv.ParseBool(vals[3])
		}

		sms = append(sms, SNMPMetrics{OIDS: vals, Metrics: mpm})
	}
	snmp.SNMPMetricsSlice = sms

	helper := mp.NewMackerelPlugin(snmp)
	helper.Tempfile = *optTempfile

	helper.Run()
}
