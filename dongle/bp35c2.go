package dongle

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/pkg/term"
)

const (
	respOk   = "OK"
	respFail = "FAIL "
)

var (
	// ErrFailER06 represents bad request error.
	ErrFailER06 = errors.New("ER06")
)

// NewBP35C2 returns command wrapper for BP35C2.
func NewBP35C2(serialDevice string, baudrate int, readTimeout time.Duration) *BP35C2 {
	return &BP35C2{
		serialDevice: serialDevice,
		baudrate:     baudrate,
		readTimeout:  readTimeout,
	}
}

// BP35C2 is BP35C0/BP35C2 client implementation.
type BP35C2 struct {
	m            sync.Mutex
	serialDevice string
	baudrate     int
	readTimeout  time.Duration
	term         *term.Term
	localIP      string
	remoteIP     string
}

// SetIP sets ipv6 address of client.
func (d *BP35C2) SetIP(ip string) {
	d.localIP = ip
}

// SetRemoteIP sets ipv6 address of remote.
func (d *BP35C2) SetRemoteIP(ip string) {
	d.remoteIP = ip
}

// Connect connects serial device.
func (d *BP35C2) Connect() error {
	t, err := term.Open(d.serialDevice,
		term.Speed(d.baudrate), term.RawMode, term.ReadTimeout(d.readTimeout))
	if err != nil {
		return err
	}
	d.term = t
	return d.clear()
}

func (d *BP35C2) clear() error {
	// Discard data before first command.
	if err := d.term.Flush(); err != nil {
		return err
	}
	if _, err := io.Copy(ioutil.Discard, d.term); err != nil {
		return err
	}
	if err := d.SKTERM(); err != nil {
		return err
	}
	return nil
}

// Close closes serial device.
func (d *BP35C2) Close() {
	d.SKTERM()
	d.term.Close()
}

// SKVER returns firmware version.
func (d *BP35C2) SKVER() (string, error) {
	d.m.Lock()
	defer d.m.Unlock()
	err := d.write("SKVER\r\n")
	if err != nil {
		return "", err
	}
	lines, err := d.readUntil(respOk)
	if err != nil {
		return "", err
	}
	const event = "EVER "
	for _, l := range lines {
		if strings.HasPrefix(l, event) {
			return strings.TrimPrefix(l, event), nil
		}
	}
	return "", errors.New("SKVER failed")
}

// EINFO is event data for SKINFO.
type EINFO struct {
	IP      string
	MAC     string
	Channel string
	PanID   string
	Side    string
}

// SKINFO returns network informations.
func (d *BP35C2) SKINFO() (*EINFO, error) {
	d.m.Lock()
	defer d.m.Unlock()
	err := d.write("SKINFO\r\n")
	if err != nil {
		return nil, err
	}
	lines, err := d.readUntil(respOk)
	if err != nil {
		return nil, err
	}
	const event = "EINFO "
	for _, l := range lines {
		if strings.HasPrefix(l, event) {
			info := &EINFO{}
			for i, e := range strings.Split(strings.TrimPrefix(l, event), " ") {
				switch i {
				case 0:
					info.IP = e
				case 1:
					info.MAC = e
				case 2:
					info.Channel = e
				case 3:
					info.PanID = e
				case 4:
					info.Side = e
				}
			}
			return info, nil
		}
	}
	return nil, errors.New("SKINFO failed")
}

// SKSETPWD generates PSK from pwd and register.
func (d *BP35C2) SKSETPWD(pwd string) error {
	d.m.Lock()
	defer d.m.Unlock()
	// SKSETPWD <LEN> <PWD><CRLF>
	err := d.write(fmt.Sprintf("SKSETPWD C %s\r\n", pwd))
	if err != nil {
		return err
	}
	return nil
}

// SKSETRBID generates Route-B ID from rbid and register.
func (d *BP35C2) SKSETRBID(rbid string) error {
	d.m.Lock()
	defer d.m.Unlock()
	// SKSETRBID <ID><CRLF>
	err := d.write(fmt.Sprintf("SKSETRBID %s\r\n", rbid))
	if err != nil {
		return err
	}
	return nil
}

// PAN is Personal Area Network.
type PAN struct {
	Channel     string
	ChannelPage string
	PanID       string
	Addr        string
	LQI         string
	PairID      string
}

// SKSCAN scans PAN.
func (d *BP35C2) SKSCAN() (*PAN, error) {
	d.m.Lock()
	defer d.m.Unlock()

	// SKSCAN <MODE> <CHANNEL_MASK> <DURATION> <SIDE><CRLF>
	err := d.write("SKSCAN 2 FFFFFFFF 6 0\r\n")
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(d.term)
	scanner := bufio.NewScanner(reader)
	if err := d.term.Flush(); err != nil {
		return nil, err
	}
	pan := &PAN{}
	for scanner.Scan() {
		l := scanner.Text()
		fmt.Println("skscan:", l) // DEBUG
		switch {
		case strings.Contains(l, "Channel:"):
			pan.Channel = strings.Split(l, ":")[1]
		case strings.Contains(l, "Channel Page:"):
			pan.ChannelPage = strings.Split(l, ":")[1]
		case strings.Contains(l, "Pan ID:"):
			pan.PanID = strings.Split(l, ":")[1]
		case strings.Contains(l, "Addr:"):
			pan.Addr = strings.Split(l, ":")[1]
		case strings.Contains(l, "LQI:"):
			pan.LQI = strings.Split(l, ":")[1]
		case strings.Contains(l, "PairID:"):
			pan.PairID = strings.Split(l, ":")[1]
		}
		if strings.Contains(l, "EVENT 22 ") {
			break
		}
		if strings.Contains(l, respFail) {
			return nil, fmt.Errorf("Failed to SKSCAN. %s", l)
		}
	}
	if pan.Addr == "" {
		return nil, errors.New("Failed to SKSCAN")
	}
	return pan, nil
}

// SKSREG registers virtual register.
func (d *BP35C2) SKSREG(k, v string) error {
	d.m.Lock()
	defer d.m.Unlock()
	// SKSREG <SREG> <VAL><CRLF>
	err := d.write(fmt.Sprintf("SKSREG %s %s\r\n", k, v))
	if err != nil {
		return err
	}
	_, err = d.readUntil(respOk)
	if err != nil {
		return err
	}
	return nil
}

// SKLL64 returns IPv6 link-local address from MAC address.
func (d *BP35C2) SKLL64(addr string) (string, error) {
	d.m.Lock()
	defer d.m.Unlock()
	// SKLL64 <ADDR64><CRLF>
	err := d.write(fmt.Sprintf("SKLL64 %s\r\n", addr))
	if err != nil {
		return "", err
	}
	const linkLocalPrefix = "FE80:0000:0000:0000:"
	lines, err := d.readUntil(linkLocalPrefix)
	if err != nil {
		return "", err
	}
	for _, l := range lines {
		if strings.HasPrefix(l, linkLocalPrefix) {
			return l, nil
		}
	}
	return "", errors.New("SKLL64 failed")
}

// SKJOIN starts PANA session.
func (d *BP35C2) SKJOIN(ipv6Addr string) error {
	d.m.Lock()
	defer d.m.Unlock()
	// SKJOIN <IPADDR><CRLF>
	err := d.write(fmt.Sprintf("SKJOIN %s\r\n", ipv6Addr))
	if err != nil {
		return err
	}
	reader := bufio.NewReader(d.term)
	scanner := bufio.NewScanner(reader)
	if err := d.term.Flush(); err != nil {
		return err
	}
	for scanner.Scan() {
		l := scanner.Text()
		fmt.Println(l)
		if strings.Contains(l, respFail) {
			return fmt.Errorf("Failed to SKJOIN. %s", l)
		}
		if strings.Contains(l, "EVENT 25 ") {
			break
		}
	}
	if scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	return nil
}

// SKSENDTO sends UDP data.
func (d *BP35C2) SKSENDTO(handle, ipAddr, port, sec string, data []byte) (string, error) {
	d.m.Lock()
	defer d.m.Unlock()
	// SKSENDTO <HANDLE> <IPADDR> <PORT> <SEC> <SIDE> <DATALEN> <DATA>
	s := fmt.Sprintf("SKSENDTO %s %s %s %s 0 %.4X ", handle, ipAddr, port, sec, len(data))
	b := append([]byte(s), data[:]...)
	b = append(b, []byte("\r\n")...)
	_, err := d.term.Write(b)
	if err != nil {
		return "", err
	}

	// ERXUDP <SENDER> <DEST>
	symbol := fmt.Sprintf("ERXUDP %s %s", d.remoteIP, d.localIP)
	lines, err := d.readUntil(symbol)
	if err != nil {
		return "", err
	}
	for _, l := range lines {
		if strings.Contains(l, respFail) {
			if l == "FAIL ER06" {
				return "", ErrFailER06
			}
			return "", fmt.Errorf("Failed to SKSENDTO. %s", l)
		}
		if strings.HasPrefix(l, symbol) {
			return l, nil
		}
	}
	return "", errors.New("SKSENDTO failed")
}

// SKTERM terminates PANA session.
func (d *BP35C2) SKTERM() error {
	d.m.Lock()
	defer d.m.Unlock()
	// SKJOIN<CRLF>
	err := d.write("SKTERM\r\n")
	if err != nil {
		return err
	}
	reader := bufio.NewReader(d.term)
	scanner := bufio.NewScanner(reader)
	if err := d.term.Flush(); err != nil {
		return err
	}
	for scanner.Scan() {
		l := scanner.Text()
		fmt.Println(l)
		if strings.Contains(l, respFail+"ER10") {
			// session is not established
			return nil
		}
		if strings.Contains(l, respFail) {
			return fmt.Errorf("Failed to SKTERM. %s", l)
		}
		if strings.Contains(l, "EVENT 28 ") {
			// session timeout (session is terminated)
			return nil
		}
		if strings.Contains(l, respOk) {
			return nil
		}
	}
	return nil
}

func (d *BP35C2) write(s string) error {
	_, err := d.term.Write([]byte(s))
	if err != nil {
		return err
	}
	return nil
}

func (d *BP35C2) readUntil(symbol string) ([]string, error) {
	reader := bufio.NewReader(d.term)
	scanner := bufio.NewScanner(reader)
	if err := d.term.Flush(); err != nil {
		return nil, err
	}
	rs := make([]string, 0, 1)
	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		fmt.Println("[RESPONSE] >>", l) // DEBUG
		rs = append(rs, l)
		if strings.Contains(l, "EVENT 29 ") {
			return nil, errors.New("session lifetime exceeded")
		}
		if strings.Contains(l, respFail) {
			return nil, errors.New(l)
		}
		if strings.HasPrefix(l, symbol) {
			break
		}
	}
	return rs, nil
}
