package main

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"

	"github.com/dimuls/classifier/entity"
)

const SourceName = "lenta.ru"

type DocumentsLoader struct {
	log *logrus.Entry
}

func NewDocumentsLoader() *DocumentsLoader {
	return &DocumentsLoader{
		log: logrus.WithField("subsystem", "documents_loader"),
	}
}

func (dl *DocumentsLoader) Load(rubrics []string,
	docsPerRubric int, from time.Time) []entity.Document {

	var (
		docs []entity.Document
		mx   sync.Mutex
		wg   sync.WaitGroup
	)

	for _, r := range rubrics {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			rubricDocs, err := dl.loadRubric(r, docsPerRubric, from)
			if err != nil {
				dl.log.WithError(err).WithField("rubric", r).
					Error("failed to load docs for rubric")
				return
			}
			mx.Lock()
			docs = append(docs, rubricDocs...)
			mx.Unlock()
		}(r)
	}

	wg.Wait()

	return docs
}

func (dl *DocumentsLoader) loadRubric(rubric string, count int,
	date time.Time) ([]entity.Document, error) {

	var docs []entity.Document

	for i := 0; i < 30; i++ {
		newDocs, err := dl.articles(rubric, date, count-len(docs))
		if err != nil {
			date = date.AddDate(0, 0, -1)
			continue
		}

		docs = append(docs, newDocs...)

		if len(docs) > count {
			return docs, nil
		}

		date = date.AddDate(0, 0, -1)
	}

	return docs, nil
}

const baseURL = "https://lenta.ru"

func formArticlesURL(rubric string, date time.Time) string {
	articlesURL := baseURL + "/rubrics/" + rubric +
		"/{year}/{month}/{day}/"
	articlesURL = strings.Replace(articlesURL, "{year}",
		strconv.Itoa(date.Year()), 1)
	articlesURL = strings.Replace(articlesURL, "{month}",
		fmt.Sprintf("%02d", date.Month()), 1)
	return strings.Replace(articlesURL, "{day}",
		fmt.Sprintf("%02d", date.Day()), 1)
}

func (dl *DocumentsLoader) articles(rubric string, date time.Time, count int) (
	[]entity.Document, error) {
	asURL := formArticlesURL(rubric, date)

	log := dl.log.WithField("articles_url", asURL)

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	res, err := httpClient.Get(asURL)
	if err != nil {
		log.WithError(err).Error("failed to get articles URL")
		return nil, errors.New("failed to HTTP get articles URL: " + err.Error())
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusFound {
			return nil, nil
		}
		log.WithField("status_code", res.StatusCode).
			Error("get articles returned not OK status code")
		return nil, errors.New("not OK status code")
	}

	gq, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, errors.New("failed to parse articles HTML: " + err.Error())
	}

	var docs []entity.Document

	gq.Find(".item.news").EachWithBreak(
		func(_ int, sel *goquery.Selection) bool {
			urlPath, urlPathExists := sel.Find(".titles > h3 > a").
				Attr("href")
			if !urlPathExists {
				log.WithError(err).Warning("failed to find article URL")
				return true
			}

			text, err := dl.articleText(baseURL + urlPath)
			if err != nil {
				log.WithError(err).Warning("failed to get article text")
			}

			docs = append(docs, entity.Document{
				Class: rubric,
				Text:  text,
			})

			return len(docs) < count
		})

	return docs, nil
}

func (dl *DocumentsLoader) articleText(aURL string) (string, error) {
	res, err := http.Get(aURL)
	if err != nil {
		return "", errors.New("failed to HTTP get article URL: " + err.Error())
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", errors.New("not OK status code")
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", errors.New("failed to parse article HTML: " + err.Error())
	}

	var ps []string

	doc.Find(".b-text > p").Each(func(i int, s *goquery.Selection) {
		ps = append(ps,
			strings.TrimSpace(html.UnescapeString(s.Text())))
	})

	return strings.Join(ps, " "), nil
}
