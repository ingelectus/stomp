package stomp

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/stomp.v2/frame"
)

// ConnOptions is an opaque structure used to collection options
// for connecting to the other server.
type connOptions struct {
	FrameCommand    string
	Host            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	HeartBeatError  time.Duration
	Login, Passcode string
	AcceptVersions  []string
	Header          *frame.Header
}

func newConnOptions(conn *Conn, opts []func(*Conn) error) (*connOptions, error) {
	co := &connOptions{
		FrameCommand:   frame.CONNECT,
		ReadTimeout:    time.Minute,
		WriteTimeout:   time.Minute,
		HeartBeatError: DefaultHeartBeatError,
	}

	// This is a slight of hand, attach the options to the Conn long
	// enough to run the options functions and then detach again.
	// The reason we do this is to allow for future options to be able
	// to modify the Conn object itself, in case that becomes desirable.
	conn.options = co
	defer func() { conn.options = nil }()

	for _, opt := range opts {
		if opt == nil {
			return nil, ErrNilOption
		}
		err := opt(conn)
		if err != nil {
			return nil, err
		}
	}

	if len(co.AcceptVersions) == 0 {
		co.AcceptVersions = append(co.AcceptVersions, string(V10), string(V11), string(V12))
	}

	return co, nil
}

func (co *connOptions) NewFrame() (*frame.Frame, error) {
	f := frame.New(co.FrameCommand)
	if co.Host != "" {
		f.Header.Set(frame.Host, co.Host)
	}

	// heart-beat
	{
		send := co.WriteTimeout / time.Millisecond
		recv := co.ReadTimeout / time.Millisecond
		f.Header.Set(frame.HeartBeat, fmt.Sprintf("%d,%d", send, recv))
	}

	// login, passcode
	if co.Login != "" || co.Passcode != "" {
		f.Header.Set(frame.Login, co.Login)
		f.Header.Set(frame.Passcode, co.Passcode)
	}

	// accept-version
	f.Header.Set(frame.AcceptVersion, strings.Join(co.AcceptVersions, ","))

	// custom header entries -- note that these do not override
	// header values already set as they are added to the end of
	// the header array
	f.Header.AddHeader(co.Header)

	return f, nil
}

// Options for connecting to the STOMP server. Used with the
// stomp.Dial and stomp.Connect functions, both of which have examples.
var ConnOpt struct {
	// Login is a connect option that allows the calling program to
	// specify the "login" and "passcode" values to send to the STOMP
	// server.
	Login func(login, passcode string) func(*Conn) error

	// Host is a connect option that allows the calling program to
	// specify the value of the "host" header.
	Host func(host string) func(*Conn) error

	// UseStomp is a connect option that specifies that the client
	// should use the "STOMP" command instead of the "CONNECT" command.
	// Note that using "STOMP" is only valid for STOMP version 1.1 and later.
	UseStomp func(*Conn) error

	// AcceptVersoin is a connect option that allows the client to
	// specify one or more versions of the STOMP protocol that the
	// client program is prepared to accept. If this option is not
	// specified, the client program will accept any of STOMP versions
	// 1.0, 1.1 or 1.2.
	AcceptVersion func(versions ...Version) func(*Conn) error

	// HeartBeat is a connect option that allows the client to specify
	// the send and receive timeouts for the STOMP heartbeat negotiation mechanism.
	// The sendTimeout parameter specifies the maximum amount of time
	// between the client sending heartbeat notifications from the server.
	// The recvTimeout paramter specifies the minimum amount of time between
	// the client expecting to receive heartbeat notifications from the server.
	// If not specified, this option defaults to one minute for both send and receive
	// timeouts.
	HeartBeat func(sendTimeout, recvTimeout time.Duration) func(*Conn) error

	// HeartBeatError is a connect option that will normally only be specified during
	// testing. It specifies a short time duration that is larger than the amount of time
	// that will take for a STOMP frame to be transmitted from one station to the other.
	// When not specified, this value defaults to 5 seconds. This value is set to a much
	// shorter time duration during unit testing.
	HeartBeatError func(errorTimeout time.Duration) func(*Conn) error

	// Header is a connect option that allows the client to specify custom header
	// elements in the CONNECT/STOMP frame.
	Header func(header *frame.Header) func(*Conn) error
}

func init() {
	ConnOpt.Login = func(login, passcode string) func(*Conn) error {
		return func(c *Conn) error {
			c.options.Login = login
			c.options.Passcode = passcode
			return nil
		}
	}

	ConnOpt.Host = func(host string) func(*Conn) error {
		return func(c *Conn) error {
			c.options.Host = host
			return nil
		}
	}

	ConnOpt.UseStomp = func(c *Conn) error {
		c.options.FrameCommand = frame.STOMP
		return nil
	}

	ConnOpt.AcceptVersion = func(versions ...Version) func(*Conn) error {
		return func(c *Conn) error {
			for _, version := range versions {
				if err := version.CheckSupported(); err != nil {
					return err
				}
				c.options.AcceptVersions = append(c.options.AcceptVersions, string(version))
			}
			return nil
		}
	}

	ConnOpt.HeartBeat = func(sendTimeout, recvTimeout time.Duration) func(*Conn) error {
		return func(c *Conn) error {
			c.options.WriteTimeout = sendTimeout
			c.options.ReadTimeout = recvTimeout
			return nil
		}
	}

	ConnOpt.HeartBeatError = func(errorTimeout time.Duration) func(*Conn) error {
		return func(c *Conn) error {
			c.options.HeartBeatError = errorTimeout
			return nil
		}
	}

	ConnOpt.Header = func(header *frame.Header) func(*Conn) error {
		return func(c *Conn) error {
			c.options.Header = header
			return nil
		}
	}
}