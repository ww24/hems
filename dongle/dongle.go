package dongle

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	readTimeout = 5 * time.Second
)

var (
	// ErrFatal is not retriable error.
	ErrFatal = errors.New("Fatal error")
)

// New returns dongle wrapper.
func New(dongle Dongle, opts ...Option) *Client {
	d := &Client{
		Logger: zap.NewNop(),
		Dongle: dongle,
	}
	d.applyOptions(opts)
	return d
}

// Client represents abstract layer for dongle.
type Client struct {
	Logger   *zap.Logger
	Dongle   Dongle
	Ipv6addr string
	rbID     string
	pwd      string
}

// Dongle represents command wrapper.
type Dongle interface {
	Connect() error
	Close()

	SKVER() (string, error)
	SKINFO() (*EINFO, error)
	SetIP(ip string)
	SetRemoteIP(ip string)

	SKSETPWD(pwd string) error
	SKSETRBID(rbid string) error
	SKSCAN() (*PAN, error)
	SKSREG(k, v string) error
	SKLL64(addr string) (string, error)
	SKJOIN(ipv6Addr string) error
	SKSENDTO(handle, ipAddr, port, sec string, data []byte) (string, error)
}

func (du *Client) Init() error {
	logger := du.Logger

	logger.Info("Connect...")
	if err := du.Dongle.Connect(); err != nil {
		logger.Error("Connect.")
		return err
	}
	logger.Info("Connect OK.")

	logger.Info("Wait 1sec...")
	time.Sleep(time.Second * 1)
	logger.Info("Wait complete.")

	logger.Info("SKVER...")
	v, err := du.Dongle.SKVER()
	logger.Info(fmt.Sprintf("SKVER Response : %s", v))
	if err != nil {
		logger.Error("SKVER.")
		return err
	}
	logger.Info("SKVER OK.")

	logger.Info("SKINFO...")
	info, err := du.Dongle.SKINFO()
	if err != nil {
		logger.Error("SKINFO.")
		return err
	}
	logger.Info("SKINFO OK.")
	du.Dongle.SetIP(info.IP)

	err = du.Dongle.SKSETPWD(du.pwd)
	if err != nil {
		logger.Error("SKSETPWD.")
		return err
	}

	err = du.Dongle.SKSETRBID(du.rbID)
	if err != nil {
		logger.Error("SKSETRBID.")
		return err
	}

	pan, err := du.Dongle.SKSCAN()
	fmt.Printf("%#v\n", pan)
	if err != nil {
		logger.Error("Failed to SKSCAN.")
		return err
	}

	err = du.Dongle.SKSREG("S2", pan.Channel)
	if err != nil {
		logger.Error("SKSREG S2.")
		return err
	}

	fmt.Println("Set PanID to S3 register...")
	err = du.Dongle.SKSREG("S3", pan.PanID)
	if err != nil {
		logger.Error("SKSREG S3.")
		return err
	}
	fmt.Println("Get IPv6 Addr with SKLL64...")
	ipv6Addr, err := du.Dongle.SKLL64(pan.Addr)
	du.Ipv6addr = ipv6Addr
	du.Dongle.SetRemoteIP(ipv6Addr)
	if err != nil {
		logger.Error("Failed to get IPv6 Address.", zap.Error(err))
		return err
	}

	fmt.Println("IPv6 Addr is " + ipv6Addr)
	fmt.Println("SKJOIN...")
	err = du.Dongle.SKJOIN(ipv6Addr)
	if err != nil {
		logger.Error("SKJOIN.")
		return err
	}

	return nil
}

func (du *Client) Close() {
	du.Dongle.Close()
}

func (du *Client) Fetch(ctx context.Context, f func(time time.Time, watt int64)) error {
	logger := du.Logger

	logger.Info("SKSENDTO...")
	done := make(chan error, 1)
	go func() {
		t, w, err := du.fetch()
		if err != nil {
			done <- err
			return
		}
		f(t, w) // callback
		done <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (du *Client) fetch() (time.Time, int64, error) {
	t := time.Now()
	logger := du.Logger

	// Send ECHONET Lite command.
	var b = []byte{0x10, 0x81, 0x00, 0x01, 0x05, 0xFF, 0x01, 0x02, 0x88, 0x01, 0x62, 0x01, 0xE7, 0x00}
	r, err := du.Dongle.SKSENDTO("1", du.Ipv6addr, "0E1A", "1", b)
	if err != nil {
		return t, 0, err
	}
	a := strings.Split(r, " ")
	if len(a) != 10 {
		return t, 0, errors.New("unexpected response")
	}
	if a[8] != "0012" {
		fmt.Println(fmt.Sprintf("%s is not 0012. ", a[7]))
		return t, 0, err
	}
	o := a[9]
	w, err := strconv.ParseInt(o[len(o)-8:], 16, 0)
	if err != nil {
		return t, 0, err
	}
	logger.Info(fmt.Sprintf("%+v", t) + " : " + fmt.Sprintf("%d", w))
	return t, w, nil
}
