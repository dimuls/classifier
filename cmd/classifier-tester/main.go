package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/dimuls/classifier/entity"
)

func main() {
	var (
		classifierURI string
		reload        bool
		rubrics       []string
		docsPerRubric uint
		docsFilePath  string
		testFromStr   string
	)

	pflag.StringVar(&classifierURI, "classifier-uri",
		"http://localhost:80", "classifier base URI")

	pflag.BoolVar(&reload, "reload", false,
		"documents reload required")

	pflag.StringSliceVar(&rubrics, "rubrics", nil,
		"rubrics required to load")

	pflag.UintVar(&docsPerRubric, "docs-per-rubric", 100,
		"docs per rubric count will be loaded")

	pflag.StringVar(&docsFilePath, "docs-file", "documents.json",
		"loaded documents file path")

	pflag.StringVar(&testFromStr, "test-from", "",
		"date from which testing docs will be loaded")

	pflag.Parse()

	if testFromStr != "" {
		if len(rubrics) == 0 {
			logrus.Fatal("rubrics should be specified")
		}

		testFrom, err := time.Parse("2006-01-02", testFromStr)
		if err != nil {
			logrus.WithError(err).Fatal("failed to parse test from date")
		}

		docs := NewDocumentsLoader().Load(rubrics, int(docsPerRubric), testFrom)
		if len(docs) != len(rubrics)*int(docsPerRubric) {
			logrus.WithFields(logrus.Fields{
				"docs_loaded":   len(docs),
				"docs_required": len(rubrics) * int(docsPerRubric),
			}).Warning("required number of docs not loaded")
		}

		testUsingDocs(classifierURI, docs)
		return
	}

	docsFileStat, err := os.Stat(docsFilePath)

	docsFileExists := !os.IsNotExist(err)

	if docsFileExists && docsFileStat.IsDir() {
		logrus.Fatal("docs file path is a directory")
	}

	var docs []entity.Document

	if reload || !docsFileExists {
		if len(rubrics) == 0 {
			logrus.WithFields(logrus.Fields{
				"reload":           reload,
				"docs_file_exists": docsFileExists,
			}).Fatal("rubrics should be specified")
		}

		docs = NewDocumentsLoader().
			Load(rubrics, int(docsPerRubric), time.Now())
		if len(docs) != len(rubrics)*int(docsPerRubric) {
			logrus.WithFields(logrus.Fields{
				"docs_loaded":   len(docs),
				"docs_required": len(rubrics) * int(docsPerRubric),
			}).Warning("required number of docs not loaded")
		}

		logStats(docs)

		f, err := os.Create(docsFilePath)
		if err != nil {
			logrus.WithError(err).Fatal("failed to create docs file")
		}

		err = json.NewEncoder(f).Encode(docs)
		if err != nil {
			logrus.WithError(err).Fatal("failed to encode docs to file")
		}
	} else {
		f, err := os.Open(docsFilePath)
		if err != nil {
			logrus.WithError(err).Fatal("failed to open docs file")
		}

		err = json.NewDecoder(f).Decode(&docs)
		if err != nil {
			logrus.WithError(err).Fatal("failed to decode docs from file")
		}
	}

	docsJSON, err := json.Marshal(docs)
	if err != nil {
		logrus.WithError(err).Fatal("failed to JSON marshal docs")
	}

	res, err := http.Post(classifierURI+"/train", "application/json",
		bytes.NewReader(docsJSON))
	if err != nil {
		logrus.WithError(err).Fatal("failed to start training classifier")
	}

	if res.StatusCode != http.StatusAccepted {
		logrus.WithField("status_code", res.StatusCode).
			Fatal("not 202 (accepted) response code")
	}
}

func logStats(docs []entity.Document) {
	classesMap := map[string]int{}
	docsLengthMap := map[string]int{}

	for _, d := range docs {
		classesMap[d.Class]++
		docsLengthMap[d.Class] += len(d.Text)
	}

	for c, count := range classesMap {
		docsTotalLength := docsLengthMap[c]
		logrus.WithFields(logrus.Fields{
			"class":               c,
			"average_text_length": float64(docsTotalLength) / float64(count),
		}).Info("class stats")
	}
}

func testUsingDocs(classifierURI string, docs []entity.Document) {
	totalErrors := 0

	total := map[string]int{}
	errors := map[string]int{}

	classifyURI := classifierURI + "/classify"

	type document struct {
		Text string
	}

	for _, d := range docs {
		dJSON, err := json.Marshal(document{Text: d.Text})
		if err != nil {
			logrus.WithError(err).
				Fatal("failed to JSON marshal test document")
		}

		res, err := http.Post(classifyURI, "application/json",
			bytes.NewReader(dJSON))
		if err != nil {
			logrus.WithError(err).Fatal("failed to post classify")
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			logrus.WithField("status_code", res.StatusCode).
				Fatal("not OK status code")
		}

		var predictedClass string

		err = json.NewDecoder(res.Body).Decode(&predictedClass)
		if err != nil {
			logrus.WithError(err).Error("failed to decode response body")
		}

		total[d.Class]++

		if d.Class != predictedClass {
			errors[d.Class]++
			totalErrors++
		}
	}

	classesMap := map[string]int{}
	for _, d := range docs {
		classesMap[d.Class]++
	}

	for c := range classesMap {
		logrus.WithFields(logrus.Fields{
			"class":      c,
			"total_docs": total[c],
			"errors":     errors[c],
			"fail_rate":  float64(errors[c]) / float64(total[c]) * 100,
		}).Info("class test stats")
	}

	logrus.WithFields(logrus.Fields{
		"total_docs":      len(docs),
		"total_errors":    totalErrors,
		"total_fail_rate": float64(totalErrors) / float64(len(docs)) * 100,
	}).Info("total stats")
}
