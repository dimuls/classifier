package web

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sirupsen/logrus"

	"github.com/dimuls/classifier/entity"
)

type Classifier interface {
	Train(docs []entity.Document)
	Trained() bool
	Classify(doc string) (string, error)
}

type Server struct {
	bindAddr   string
	debug      bool
	classifier Classifier

	echo *echo.Echo

	training int32

	waitGroup sync.WaitGroup

	log *logrus.Entry
}

func NewServer(bindAddr string, c Classifier, debug bool) *Server {

	return &Server{
		bindAddr:   bindAddr,
		debug:      debug,
		classifier: c,

		log: logrus.WithField("subsystem", "web_server"),
	}
}

func (s *Server) Start() {
	e := echo.New()

	e.Debug = s.debug

	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.Recover())
	e.Use(logrusLogger)

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		var (
			code = http.StatusInternalServerError
			msg  interface{}
		)

		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			msg = he.Message
		} else if e.Debug {
			msg = err.Error()
		} else {
			msg = http.StatusText(code)
		}
		if _, ok := msg.(string); !ok {
			msg = fmt.Sprintf("%v", msg)
		}

		// Send response
		if !c.Response().Committed {
			if c.Request().Method == http.MethodHead { // Issue #608
				err = c.NoContent(code)
			} else {
				err = c.String(code, msg.(string))
			}
			if err != nil {
				s.log.WithError(err).Error("failed to error response")
			}
		}
	}

	e.POST("/train", s.postTrain)
	e.POST("/classify", s.postClassify)
	e.GET("/training", s.getTraining)

	s.echo = e

	s.waitGroup.Add(1)
	go func() {
		defer s.waitGroup.Done()
		err := e.Start(s.bindAddr)
		if err != nil && err != http.ErrServerClosed {
			s.log.WithError(err).Error("failed to start")
		}
	}()
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	err := s.echo.Shutdown(ctx)
	if err != nil {
		s.log.WithError(err).Error("failed to graceful stop")
	}

	s.waitGroup.Wait()
}

func logrusLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()

		err := next(c)

		stop := time.Now()

		if err != nil {
			c.Error(err)
		}

		req := c.Request()
		res := c.Response()

		p := req.URL.Path
		if p == "" {
			p = "/"
		}

		bytesIn := req.Header.Get(echo.HeaderContentLength)
		if bytesIn == "" {
			bytesIn = "0"
		}

		entry := logrus.WithFields(map[string]interface{}{
			"subsystem":    "web_server",
			"remote_ip":    c.RealIP(),
			"host":         req.Host,
			"query_params": c.QueryParams(),
			"uri":          req.RequestURI,
			"method":       req.Method,
			"path":         p,
			"referer":      req.Referer(),
			"user_agent":   req.UserAgent(),
			"status":       res.Status,
			"latency":      stop.Sub(start).String(),
			"bytes_in":     bytesIn,
			"bytes_out":    strconv.FormatInt(res.Size, 10),
		})

		const msg = "request handled"

		if res.Status >= 500 {
			if err != nil {
				entry = entry.WithError(err)
			}
			entry.Error(msg)
		} else if res.Status >= 400 {
			entry.Warn(msg)
		} else {
			entry.Info(msg)
		}

		return nil
	}
}
