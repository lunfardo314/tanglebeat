package main

import (
	"fmt"
	"github.com/lunfardo314/tanglebeat/tanglebeat/senderpart"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strings"
)

func runWebServer(port int) {
	infof("Web server for Prometheus metrics and debug dashboard will be running on port '%d'", port)
	http.HandleFunc("/loadjs", loadjsHandler)
	http.HandleFunc("/dashboard", dashboardHandler)
	http.HandleFunc("/api1/internal_stats/", internalStatsHandler)
	http.HandleFunc("/api1/conf_time", senderpart.HandlerConfStats)
	http.HandleFunc("/api1/senders", senderpart.HandlerSenderStates)
	http.Handle("/metrics", promhttp.Handler())
	panic(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func internalStatsHandler(w http.ResponseWriter, r *http.Request) {
	req := r.URL.Path[len("/api1/internal_stats/"):]
	_, _ = fmt.Fprintf(w, string(getGlbStatsJSON(strings.HasPrefix(req, "formatted"))))
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprint(w, indexPage)
}

func loadjsHandler(w http.ResponseWriter, r *http.Request) {
	_, _ = fmt.Fprint(w, loadjs)
}
