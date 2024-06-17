package pipeline

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/api"
	"github.com/prometheus/statsd_exporter/pkg/level"
)

type requestFilterRoundTripper struct {
	next       http.RoundTripper
	blockRegex []*regexp.Regexp
	logger     log.Logger
}

// NewRequestFilterWare looks at queries in request and immediately
// sends a 400 response if query matches a regex
func NewRequestFilterWare(logger log.Logger, blockRegexList []*regexp.Regexp) Middleware {
	return MiddlewareFunc(func(next http.RoundTripper) http.RoundTripper {

		if len(blockRegexList) == 0 {
			return next
		}

		return requestFilterRoundTripper{
			next:       next,
			blockRegex: blockRegexList,
			logger:     logger,
		}
	})
}

func (t requestFilterRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	searchReq, err := api.ParseSearchRequest(req)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	for _, regex := range t.blockRegex {
		if regex.Match([]byte(searchReq.Query)) {

			level.Debug(t.logger).Log("msg", "query matches regex", regex.String(), "returning 400")

			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("Query is temporarily blocked by your administrator.")),
			}, nil
		}
	}

	return t.next.RoundTrip(req)
}
