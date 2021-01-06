package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dimuls/classifier/entity"
	"github.com/labstack/echo"
)

func (s *Server) postClassifier(c echo.Context) error {
	classifier, _ := s.classifiersPool.Classifier(
		c.Param("classifier_id"), true)

	if classifier.Training() {
		return echo.NewHTTPError(http.StatusServiceUnavailable,
			"classifier is training")
	}

	var docs []entity.Document

	err := json.NewDecoder(c.Request().Body).Decode(&docs)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			"JSON decode request body: %w", err)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err = classifier.Train(docs)
		if err != nil {
			s.log.WithError(err).Error("failed to train classifier")
		}
	}()

	return c.NoContent(http.StatusAccepted)
}

func (s *Server) getClassifierTraining(c echo.Context) error {
	classifier, exists := s.classifiersPool.Classifier(
		c.Param("classifier_id"), false)
	if !exists {
		return echo.NewHTTPError(http.StatusNotFound,
			"classifier with given id not found")
	}
	return c.JSON(http.StatusOK, classifier.Training())
}

func (s *Server) postClassifierClassify(c echo.Context) error {
	classifier, exists := s.classifiersPool.Classifier(
		c.Param("classifier_id"), false)
	if !exists {
		return echo.NewHTTPError(http.StatusNotFound,
			"classifier with given id not found")
	}

	if classifier.Training() {
		return echo.NewHTTPError(http.StatusServiceUnavailable,
			"classifier is training")
	}

	var doc struct {
		Text string
	}

	err := json.NewDecoder(c.Request().Body).Decode(&doc)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			"JSON decode request body: "+err.Error())
	}

	if doc.Text == "" {
		return echo.NewHTTPError(http.StatusBadRequest,
			"empty text")
	}

	class, err := classifier.Classify(doc.Text)
	if err != nil {
		return fmt.Errorf("classify: %w", err)
	}

	return c.JSON(http.StatusOK, class)
}

func (s *Server) deleteClassifier(c echo.Context) error {
	err := s.classifiersPool.RemoveClassifier(c.Param("classifier_id"))
	if err != nil {
		return fmt.Errorf(
			"remove classifier form classifiers pool: %w", err)
	}
	return c.NoContent(http.StatusOK)
}
