package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var PobQueueGauge = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "bpl_pob_queue_size",
		Help: "Current size of the character queue to be processed by the pob server",
	},
)

var PobsCalculatedCounter = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "bpl_pobs_calculated",
		Help: "Number of PoBs calculated",
	},
)
var PobsCalculatedErrorCounter = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "bpl_pobs_calculated_error_total",
		Help: "Number of PoB calculation errors",
	},
)
var PobsSavedCounter = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "bpl_pobs_saved",
		Help: "Number of PoBs saved to the database",
	},
)
var PobsSaveErrorCounter = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "bpl_pobs_save_error_total",
		Help: "Number of PoBs that failed to be saved to the database",
	},
)
var PobsSkippedCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "bpl_pobs_skipped_total",
		Help: "Number of calculated PoBs not persisted, by reason",
	},
	[]string{"reason"},
)
var PoeRequestCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "poe_request_total",
	Help: "The total number of requests by endpoint to the PoE API",
}, []string{"endpoint"})

var ResponseCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "poe_response_total",
	Help: "The total number of responses by status code from the PoE API",
}, []string{"status_code"})

var RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "poe_request_duration_seconds",
	Help: "Duration of requests to the PoE API",
}, []string{"endpoint"})

var StashCounterTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "stash_counter_total",
	Help: "The total number of stashes processed",
})

var StashCounterFiltered = promauto.NewCounter(prometheus.CounterOpts{
	Name: "stash_counter_filtered",
	Help: "The total number of stashes filtered",
})

var ChangeIdGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "change_id",
	Help: "The current change id",
})

var NinjaChangeIdGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "ninja_change_id",
	Help: "The current change id from the poe.ninja api",
})

var TeamMatchesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "team_matches_total",
	Help: "The number of matches for each team",
}, []string{"team"})

var QueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "sql_query_duration_seconds",
	Help: "Duration of sql queries in seconds",
}, []string{"query"})

var ScoreAggregationDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "score_aggregation_duration_s",
	Help: "Duration of Aggregation step during scoring",
}, []string{"aggregation-step"})

var ScoreEvaluationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "score_evaluation_duration_s",
	Help: "Duration of Evaluation step during scoring",
	Buckets: []float64{
		0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10,
	},
})
