package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/ww24/hems/dongle"
	"github.com/ww24/hems/metric"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
)

const (
	initTimeout   = 30 * time.Second
	fetchTimeout  = 10 * time.Second
	fetchWaitTime = 10 * time.Second
	maxRetryCount = 5
	metricsPort   = 9999
)

var (
	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(os.Stderr),
		zapcore.InfoLevel,
	))

	metrics = expvar.NewMap("hems")
	watt    = new(expvar.Int)

	errMaxRetryCountExeeded = errors.New("max retry count exceeded")
)

func init() {
	metrics.Set("watt", watt)
}

func main() {
	rbID := os.Getenv("HEMS_ROUTEB_ID")
	pwd := os.Getenv("HEMS_PASSWORD")

	if rbID == "" {
		logger.Fatal("HEMS_ROUTEB_ID must be specified")
	}
	if pwd == "" {
		logger.Fatal("HEMS_PASSWORD must be specified")
	}

	logger.Info("# Started")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		retryCount := 0
		for {
			err := start(ctx, rbID, pwd)
			switch err {
			case errMaxRetryCountExeeded:
				retryCount = 0
			default:
				retryCount++
				logger.Warn("failed to start", zap.Int("retryCount", retryCount))
				if retryCount > maxRetryCount {
					logger.Fatal("max retry count exceeded, reboot...")
				}
			}
		}
	}()

	go metric.SyncMetrics(ctx, 30*time.Second)

	srv := &http.Server{Addr: ":" + strconv.Itoa(metricsPort)}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Error("failed to listen and serve", zap.Error(err))
			cancel()
		}
	}()

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown", zap.Error(err))
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("context finished")
	case s := <-sig:
		close(sig)
		logger.Info("Signal received", zap.String("signal", s.String()))
	}
}

func start(ctx context.Context, rbID, pwd string) error {
	var serialDevice string
	switch runtime.GOOS {
	case "darwin":
		// mac (ポートによって device が変わる)
		serialDevice = "/dev/tty.usbmodem14101"
		// d.SerialDevice = "/dev/tty.usbmodem14201"
	default:
		// raspberry pi.
		serialDevice = "/dev/ttyACM0"
	}
	d := dongle.NewBP35C2(serialDevice, 115200)
	du := dongle.New(d,
		dongle.NewLogger(logger),
		dongle.NewRbID(rbID),
		dongle.NewPwd(pwd))
	defer du.Close()

	g, c := errgroup.WithContext(ctx)
	g.Go(func() error { return du.Init() })
	go g.Wait()

	select {
	case <-c.Done():
		if err := g.Wait(); err != nil {
			logger.Error("failed to init", zap.Error(err))
			return err
		}
	case <-time.After(initTimeout):
		logger.Error("failed to init becase timeout exceeded")
		return errors.New("failed to init becase timeout exceeded")
	}

	logger.Info("# Scanned")
	return processor(ctx, du, callback)
}

// Result represents hems device response.
type Result struct {
	Watt int64
	Time time.Time
}

func callback(res *Result) {
	logger.Info("Output", zap.Int64("watt", res.Watt))
	watt.Set(int64(res.Watt))
}

func processor(ctx context.Context, du *dongle.Client, callback func(res *Result)) (err error) {
	retryCount := 0
	defer func() {
		if cause := recover(); cause != nil {
			e, ok := cause.(error)
			if !ok {
				e = fmt.Errorf("Recovered: %+v", err)
			}
			err = e
		}
	}()
	for {
		timer := time.After(fetchWaitTime)
		select {
		case <-ctx.Done():
			return
		case <-timer:
			err = func() error {
				ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
				defer cancel()
				err := du.Fetch(ctx, func(time time.Time, watt int64) {
					callback(&Result{Time: time, Watt: watt})
				})
				switch err {
				case nil:
					retryCount = 0
					return nil
				case dongle.ErrFailER06:
					logger.Error("Fatal error occurred", zap.Error(err))
					return err
				default:
					retryCount++
					logger.Error("failed to fetch",
						zap.Error(err),
						zap.Int("retryCount", retryCount),
					)
					if retryCount > maxRetryCount {
						return errMaxRetryCountExeeded
					}
					return nil
				}
			}()
			if err != nil {
				return
			}
		}
	}
}
