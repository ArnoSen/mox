package httphandler

import "net/http"

type EventSourceHandler struct {
}

func NewEventSourceHandler() EventSourceHandler {
	return EventSourceHandler{}
}

func (eh EventSourceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	AddCORSAllowedOriginHeader(w, r)
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
