package metrics

var MetricsUnderTransition = map[string]string{
	"container_cpu_utilization": "container_cpu_usage",
}

var DotMetricsUnderTransition = map[string]string{
	"container.cpu.utilization": "container.cpu.usage",
}

func GetTransitionedMetric(metric string, normalized bool) string {
	if normalized {
		if _, ok := MetricsUnderTransition[metric]; ok {
			return MetricsUnderTransition[metric]
		}
		return metric
	} else {
		if _, ok := DotMetricsUnderTransition[metric]; ok {
			return DotMetricsUnderTransition[metric]
		}
		return metric
	}
}
