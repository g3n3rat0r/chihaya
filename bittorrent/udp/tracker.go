// Copyright 2016 Jimmy Zelinskie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package udp implements a BitTorrent tracker via the UDP protocol as
// described in BEP 15.
package udp

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	"github.com/jzelinskie/trakr/bittorrent"
)

var promResponseDurationMilliseconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "trakr_udp_response_duration_milliseconds",
		Help:    "The duration of time it takes to receive and write a response to an API request",
		Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
	},
	[]string{"action", "error"},
)

// recordResponseDuration records the duration of time to respond to a UDP
// Request in milliseconds .
func recordResponseDuration(action, err error, duration time.Duration) {
	promResponseDurationMilliseconds.
		WithLabelValues(action, err.Error()).
		Observe(float64(duration.Nanoseconds()) / float64(time.Millisecond))
}

// Config represents all of the configurable options for a UDP BitTorrent
// Tracker.
type Config struct {
	Addr            string
	PrivateKey      string
	AllowIPSpoofing bool
}

// Tracker holds the state of a UDP BitTorrent Tracker.
type Tracker struct {
	sock    *net.UDPConn
	closing chan struct{}
	wg      sync.WaitGroup

	bittorrent.TrackerFuncs
	Config
}

// NewTracker allocates a new instance of a Tracker.
func NewTracker(funcs bittorrent.TrackerFuncs, cfg Config) {
	return &Tracker{
		closing:      make(chan struct{}),
		TrackerFuncs: funcs,
		Config:       cfg,
	}
}

// Stop provides a thread-safe way to shutdown a currently running Tracker.
func (t *Tracker) Stop() {
	close(t.closing)
	t.sock.SetReadDeadline(time.Now())
	t.wg.Wait()
}

// ListenAndServe listens on the UDP network address t.Addr and blocks serving
// BitTorrent requests until t.Stop() is called or an error is returned.
func (t *Tracker) ListenAndServe() error {
	udpAddr, err := net.ResolveUDPAddr("udp", t.Addr)
	if err != nil {
		return err
	}

	t.sock, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	defer t.sock.Close()

	pool := bytepool.New(256, 2048)

	for {
		// Check to see if we need to shutdown.
		select {
		case <-t.closing:
			t.wg.Wait()
			return nil
		default:
		}

		// Read a UDP packet into a reusable buffer.
		buffer := pool.Get()
		t.sock.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := t.sock.ReadFromUDP(buffer)
		if err != nil {
			pool.Put(buffer)
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				// A temporary failure is not fatal; just pretend it never happened.
				continue
			}
			return err
		}

		// We got nothin'
		if n == 0 {
			pool.Put(buffer)
			continue
		}

		log.Println("Got UDP Request")
		t.wg.Add(1)
		go func(start time.Time) {
			defer t.wg.Done()
			defer pool.Put(buffer)

			// Handle the request.
			start := time.Now()
			response, action, err := t.handleRequest(&Request{buffer[:n], addr.IP})
			log.Printf("Handled UDP Request: %s, %s, %s\n", response, action, err)

			// Record to the duration of time used to respond to the request.
			var errString string
			if err != nil {
				errString = err.Error()
			}
			recordResponseDuration(action, errString, time.Since(start))
		}()
	}
}

// Request represents a UDP payload received by a Tracker.
type Request struct {
	Packet []byte
	IP     net.IP
}

// ResponseWriter implements the ability to respond to a Request via the
// io.Writer interface.
type ResponseWriter struct {
	socket net.UDPConn
	addr   net.UDPAddr
}

// Write implements the io.Writer interface for a ResponseWriter.
func (w *ResponseWriter) Write(b []byte) (int, error) {
	w.socket.WriteToUDP(b, w.addr)
	return len(b), nil
}

// handleRequest parses and responds to a UDP Request.
func (t *Tracker) handleRequest(r *Request, w *ResponseWriter) (response []byte, actionName string, err error) {
	if len(r.packet) < 16 {
		// Malformed, no client packets are less than 16 bytes.
		// We explicitly return nothing in case this is a DoS attempt.
		err = errMalformedPacket
		return
	}

	// Parse the headers of the UDP packet.
	connID := r.packet[0:8]
	actionID := binary.BigEndian.Uint32(r.packet[8:12])
	txID := r.packet[12:16]

	// If this isn't requesting a new connection ID and the connection ID is
	// invalid, then fail.
	if actionID != connectActionID && !ValidConnectionID(connID, r.IP, time.Now(), t.PrivateKey) {
		err = errBadConnectionID
		WriteError(w, txID, err)
		return
	}

	// Handle the requested action.
	switch actionID {
	case connectActionID:
		actionName = "connect"

		if !bytes.Equal(connID, initialConnectionID) {
			err = errMalformedPacket
			return
		}

		WriteConnectionID(w, txID, NewConnectionID(r.IP, time.Now(), t.PrivateKey))
		return

	case announceActionID:
		actionName = "announce"

		var req *bittorrent.AnnounceRequest
		req, err = ParseAnnounce(r, t.AllowIPSpoofing)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		var resp *bittorrent.AnnounceResponse
		resp, err = t.HandleAnnounce(req)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		WriteAnnounce(w, txID, resp)

		if t.AfterAnnounce != nil {
			go t.AfterAnnounce(req, resp)
		}

		return

	case scrapeActionID:
		actionName = "scrape"

		var req *bittorrent.ScrapeRequest
		req, err = ParseScrape(r)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		var resp *bittorrent.ScrapeResponse
		ctx := context.TODO()
		resp, err = t.HandleScrape(ctx, req)
		if err != nil {
			WriteError(w, txID, err)
			return
		}

		WriteScrape(w, txID, resp)

		if t.AfterScrape != nil {
			go t.AfterScrape(req, resp)
		}

		return

	default:
		err = errUnknownAction
		WriteError(w, txID, err)
		return
	}
}
