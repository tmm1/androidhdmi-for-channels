package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/gin-gonic/gin"
)

func main() {
	err := run()
	if err != nil {
		panic(err)
	}
}

func run() error {
	r := gin.Default()
	r.GET("/play/tuner:tuner/:channel", func(c *gin.Context) {
		tuner := c.Param("tuner")
		channel := c.Param("channel")

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
			c.JSON(500, gin.H{"error": "invalid tuner"})
			return
		}

		if err := execute(pre); err != nil {
			log.Printf("[ERR] Failed to run pre script: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if err := execute(start, channel); err != nil {
			log.Printf("[ERR] Failed to run start script: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		resp, err := http.Get(src)
		if err != nil {
			log.Printf("[ERR] Failed to fetch source: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		} else if resp.StatusCode != 200 {
			log.Printf("[ERR] Failed to fetch source: %v", resp.Status)
			c.JSON(500, gin.H{"error": resp.Status})
			return
		}

		c.Header("Transfer-Encoding", "identity")
		c.Header("Content-Type", "video/mp2t")
		c.Writer.WriteHeaderNow()
		c.Writer.Flush()

		defer func() {
			resp.Body.Close()
			if err := execute(stop); err != nil {
				log.Printf("[ERR] Failed to run stop script: %v", err)
			}
		}()

		io.Copy(c.Writer, resp.Body)
	})
	return r.Run(":7654")
}

func execute(args ...string) error {
	log.Printf("Running %v", args)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
