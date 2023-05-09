package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	tunerLock sync.Mutex

	tuners = []tuner{
		{
			url:   "http://192.168.1.168/0.ts",
			pre:   "/opt/opendct/prebmitune.sh",
			start: "/opt/opendct/bmitune.sh",
			stop:  "/opt/opendct/stopbmitune.sh",
		},
		{
			url:   "http://192.168.1.169/main",
			pre:   "/opt/opendct/prebmituneb.sh",
			start: "/opt/opendct/bmituneb.sh",
			stop:  "/opt/opendct/stopbmituneb.sh",
		},
	}
)

type tuner struct {
	url              string
	pre, start, stop string
	active           bool
}

type reader struct {
	io.ReadCloser
	t       *tuner
	channel string
	started bool
}

func init() {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 5 * time.Second
	transport.DialContext = (&net.Dialer{
		Timeout: 5 * time.Second,
	}).DialContext
	http.DefaultClient.Transport = transport
}

func (r *reader) Read(p []byte) (int, error) {
	if !r.started {
		r.started = true
		go func() {
			if err := execute(r.t.pre); err != nil {
				log.Printf("[ERR] Failed to run pre script: %v", err)
				return
			}
			if err := execute(r.t.start, r.channel); err != nil {
				log.Printf("[ERR] Failed to run start script: %v", err)
				return
			}
		}()
	}
	return r.ReadCloser.Read(p)
}

func (r *reader) Close() error {
	if err := execute(r.t.stop); err != nil {
		log.Printf("[ERR] Failed to run stop script: %v", err)
	}
	tunerLock.Lock()
	r.t.active = false
	tunerLock.Unlock()
	return r.ReadCloser.Close()
}

func execute(args ...string) error {
	t0 := time.Now()
	log.Printf("Running %v", args)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	log.Printf("Finished running %v in %v", args[0], time.Since(t0))
	return err
}

func tune(idx, channel string) (io.ReadCloser, error) {
	tunerLock.Lock()
	defer tunerLock.Unlock()

	var t *tuner
	log.Printf("tune for %v %v", idx, channel)
	if idx == "" || idx == "auto" {
		for i, ti := range tuners {
			if ti.active {
				continue
			}
			t = &tuners[i]
			break
		}
	} else {
		i, _ := strconv.Atoi(idx)
		if i < len(tuners) && i >= 0 {
			t = &tuners[i]
		}
	}
	if t == nil {
		return nil, fmt.Errorf("tuner not available")
	}

	resp, err := http.Get(t.url)
	if err != nil {
		log.Printf("[ERR] Failed to fetch source: %v", err)
		return nil, err
	} else if resp.StatusCode != 200 {
		log.Printf("[ERR] Failed to fetch source: %v", resp.Status)
		return nil, fmt.Errorf("invalid response: %v", resp.Status)
	}

	t.active = true
	return &reader{
		ReadCloser: resp.Body,
		channel:    channel,
		t:          t,
	}, nil
}

func run() error {
	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.GET("/play/tuner:tuner/:channel", func(c *gin.Context) {
		tuner := c.Param("tuner")
		channel := c.Param("channel")

		c.Header("Transfer-Encoding", "identity")
		c.Header("Content-Type", "video/mp2t")
		c.Writer.WriteHeaderNow()
		c.Writer.Flush()

		reader, err := tune(tuner, channel)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		defer func() {
			reader.Close()
		}()

		io.Copy(c.Writer, reader)
	})
	return r.Run(":7654")
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}
