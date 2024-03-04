package httphandler

import (
	"net/http"
	"net/http/httputil"

	"log/slog"

	"github.com/mjl-/mox/mlog"
)

type EventSourceHandler struct {
	logger mlog.Log
}

func NewEventSourceHandler(logger mlog.Log) EventSourceHandler {
	return EventSourceHandler{
		logger: logger,
	}
}

//GET /jmap/eventsource/?types= HTTP/2.0\r\n
//Host: mail.km42.nl\r\n
//Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8\r\n
//Accept-Encoding: gzip, deflate, br\r\n
//Accept-Language: en-US,en;q=0.5\r\n
//Dnt: 1\r\n
//Sec-Fetch-Dest: document\r\n
//Sec-Fetch-Mode: navigate\r\n
//Sec-Fetch-Site: cross-site\r\n
//Sec-Gpc: 1\r\n
//Te: trailers\r\n
//Upgrade-Insecure-Requests: 1\r\n
//User-Agent: Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:122.0) Gecko/20100101 Firefox/122.0\r\n\r\n"

func (eh EventSourceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eh.logger.Debug("event source handler called")
	reqBytes, err := httputil.DumpRequest(r, true)
	if err == nil {
		eh.logger.Debug("event source request", slog.String("req", string(reqBytes)))
	}
	AddCORSAllowedOriginHeader(w, r)

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/event-stream")

	w.WriteHeader(http.StatusUnauthorized)

	/*
		for {
			select {
			case <-r.Context().Done():
				break
			case <-time.Tick(5 * time.Second):
				w.Write(nil)
			}
		}
	*/
}

/*


	eventSourceREStr := strings.ReplaceAll(eventSourcePath, datatyper.UrlTemplateTypes, "(\\S+)")
	eventSourceREStr = strings.ReplaceAll(eventSourceREStr, datatyper.UrlTemplateClosedAfter, "(\\d+)")
	eventSourceREStr = strings.ReplaceAll(eventSourceREStr, datatyper.UrlTemplatePing, "(\\d+)")
	eventSourceREStr = escapeCommon(eventSourceREStr)

	if eventSourcePathRE, err := regexp.Compile(eventSourceREStr); err == nil && eventSourcePathRE.MatchString(p) {
		return handlerTypeEventSource
	}

*/
