package httphandler

import "testing"

func TestGetHandlerForPath(t *testing.T) {

	const (
		exampleDownloadPath    = "/download/{accountId}/{blobId}/{name}?accept={type}"
		exampleUploadPath      = "/upload/{accountId}/"
		exampleEventSourcePath = "/eventsource/?types={types}&closeafter={closeafter}&ping={ping}"
	)

	for _, tc := range []struct {
		testname        string
		path            string
		downloadPath    string
		uploadPath      string
		eventSourcePath string
		eHandlerType    handlerType
	}{
		{
			testname:        "valid download",
			path:            "/download/1234/4444/objectname?accept=application%2Fjson",
			downloadPath:    exampleDownloadPath,
			uploadPath:      exampleUploadPath,
			eventSourcePath: exampleEventSourcePath,
			eHandlerType:    handlerTypeDownload,
		},
		{
			testname:        "accountname with non numbers",
			path:            "/download/aaa/4444/objectname?accept=application%2Fjson",
			downloadPath:    exampleDownloadPath,
			uploadPath:      exampleUploadPath,
			eventSourcePath: exampleEventSourcePath,
			eHandlerType:    handlerTypeUndefined,
		},
		{
			testname:        "valid upload",
			path:            "/upload/12324/",
			downloadPath:    exampleDownloadPath,
			uploadPath:      exampleUploadPath,
			eventSourcePath: exampleEventSourcePath,
			eHandlerType:    handlerTypeUpload,
		},
		{
			testname:        "upload with non account name numbers",
			path:            "/upload/aaaa/",
			downloadPath:    exampleDownloadPath,
			uploadPath:      exampleUploadPath,
			eventSourcePath: exampleEventSourcePath,
			eHandlerType:    handlerTypeUndefined,
		},
		{
			testname:        "valid eventsource",
			path:            "/eventsource/?types=type1&closeafter=10&ping=1",
			downloadPath:    exampleDownloadPath,
			uploadPath:      exampleUploadPath,
			eventSourcePath: exampleEventSourcePath,
			eHandlerType:    handlerTypeEventSource,
		},
		{
			testname:        "non mathing path",
			path:            "/api/",
			downloadPath:    exampleDownloadPath,
			uploadPath:      exampleUploadPath,
			eventSourcePath: exampleEventSourcePath,
			eHandlerType:    handlerTypeUndefined,
		},
		{
			testname:        "root path",
			path:            "/",
			downloadPath:    exampleDownloadPath,
			uploadPath:      exampleUploadPath,
			eventSourcePath: exampleEventSourcePath,
			eHandlerType:    handlerTypeUndefined,
		},
	} {

		t.Run(tc.testname, func(t *testing.T) {
			handlerType := getHandlerForPath(tc.path, tc.downloadPath, tc.uploadPath, tc.eventSourcePath)
			if handlerType != tc.eHandlerType {
				t.Errorf("was expecting type %d but got %d", tc.eHandlerType, handlerType)
			}
		})

	}

}
