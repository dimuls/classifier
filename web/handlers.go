package web

import (
	"errors"
	"net/http"
	"sync/atomic"

	"github.com/labstack/echo"

	"github.com/dimuls/classifier/entity"
)

func (s *Server) postTrain(c echo.Context) error {
	if atomic.LoadInt32(&s.training) == 1 {
		return echo.NewHTTPError(http.StatusServiceUnavailable,
			"already training data")
	}

	var docs []entity.Document

	err := c.Bind(&docs)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			"failed to bind body: "+err.Error())
	}

	s.waitGroup.Add(1)
	atomic.StoreInt32(&s.training, 1)
	go func() {
		defer s.waitGroup.Done()
		defer atomic.StoreInt32(&s.training, 0)
		s.classifier.Train(docs)
	}()

	return c.NoContent(http.StatusAccepted)
}

func (s *Server) getTraining(c echo.Context) error {
	return c.JSON(http.StatusOK, atomic.LoadInt32(&s.training) == 1)
}

func (s *Server) postClassify(c echo.Context) error {
	if atomic.LoadInt32(&s.training) == 1 {
		return echo.NewHTTPError(http.StatusServiceUnavailable,
			"training data")
	}

	var doc struct {
		Text string
	}

	err := c.Bind(&doc)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			"failed to bind body: "+err.Error())
	}

	class, err := s.classifier.Classify(doc.Text)
	if err != nil {
		return errors.New("failed to classify: " + err.Error())
	}

	return c.JSON(http.StatusOK, class)
}
