package qlog_quic

import (
	"bufio"
	"context"
	"github.com/quic-go/quic-go/logging"
	"os"
	"path"
	"qperf-go/common/qlog"
	"qperf-go/common/utils"
	"strings"
)

const odcidTemplate = "{odcid}"

func substituteFilePathTemplate(filePathTemplate string, odcid logging.ConnectionID) string {
	if odcid.Len() == 0 {
		return strings.ReplaceAll(filePathTemplate, odcidTemplate, "")
	} else {
		return strings.ReplaceAll(filePathTemplate, odcidTemplate, odcid.String())
	}
}

func isFilePathTemplate(filePathTemplate string) bool {
	return strings.Contains(filePathTemplate, odcidTemplate)
}

// NewFileQlogTracer creates new connection tracers using QlogWriter.
// "{odcid}" in the filePathTemplate is replaced by the ODCID of the QUIC connection.
// If the filePathTemplate contains "{odcid}" multiple files are created.
// Otherwise, everything is written to a single file.
func NewFileQlogTracer(filePathTemplate string, config *qlog.Config) func(ctx context.Context, perspective logging.Perspective, connectionID logging.ConnectionID) logging.ConnectionTracer {
	if isFilePathTemplate(filePathTemplate) {
		// write to multiple qlog files
		return func(ctx context.Context, p logging.Perspective, odcid logging.ConnectionID) logging.ConnectionTracer {
			filePath := substituteFilePathTemplate(filePathTemplate, odcid)
			err := os.MkdirAll(path.Dir(filePath), 0700)
			if err != nil {
				panic(err)
			}
			f, err := os.Create(filePath)
			if err != nil {
				panic(err)
			}
			w := utils.NewBufferedWriteCloser(bufio.NewWriter(f), f)
			config := config.Copy()
			if odcid.Len() != 0 {
				config.ODCID = odcid.String()
			}
			config.GroupID = config.ODCID
			qlogWriter := qlog.NewQlogWriter(w, config)
			return NewConnectionTracer(qlogWriter, p, odcid, true)
		}
	} else {
		// write to single qlog file
		filePath := filePathTemplate
		err := os.MkdirAll(path.Dir(filePath), 0700)
		if err != nil {
			panic(err)
		}
		f, err := os.Create(filePath)
		if err != nil {
			panic(err)
		}
		w := utils.NewBufferedWriteCloser(bufio.NewWriter(f), f)
		qlogWriter := qlog.NewQlogWriter(w, config)
		return NewTracer(qlogWriter)
	}
}
