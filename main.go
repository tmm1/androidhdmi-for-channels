package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
)

type reader struct {
	io.ReadCloser
	pre, start, stop string
	channel          string
	started          bool
}

func (r *reader) Read(p []byte) (int, error) {
	if !r.started {
		r.started = true
		go func() {
			if err := execute(r.pre); err != nil {
				log.Printf("[ERR] Failed to run pre script: %v", err)
				return
			}
			if err := execute(r.start, r.channel); err != nil {
				log.Printf("[ERR] Failed to run start script: %v", err)
				return
			}
		}()
	}
	return r.ReadCloser.Read(p)
}

func (r *reader) Close() error {
	if err := execute(r.stop); err != nil {
		log.Printf("[ERR] Failed to run stop script: %v", err)
	}
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

func tune(tuner, channel string) (io.ReadCloser, error) {
	var src, pre, start, stop string
	switch tuner {
	case "0":
		src = "http://192.168.1.168/0.ts"
		pre = "/opt/opendct/prebmitune.sh"
		start = "/opt/opendct/bmitune.sh"
		stop = "/opt/opendct/stopbmitune.sh"
	case "1":
		src = "http://192.168.1.169/main"
		pre = "/opt/opendct/prebmituneb.sh"
		start = "/opt/opendct/bmituneb.sh"
		stop = "/opt/opendct/stopbmituneb.sh"
	default:
		return nil, fmt.Errorf("invalid tuner")
	}

	resp, err := http.Get(src)
	if err != nil {
		log.Printf("[ERR] Failed to fetch source: %v", err)
		return nil, err
	} else if resp.StatusCode != 200 {
		log.Printf("[ERR] Failed to fetch source: %v", resp.Status)
		return nil, fmt.Errorf("invalid response: %v", resp.Status)
	}

	return &reader{
		ReadCloser: resp.Body,
		channel:    channel,
		pre:        pre,
		start:      start,
		stop:       stop,
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
