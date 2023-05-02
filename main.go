package main

import (
	"io"
	"net/http"
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

		exec.Command(pre).Run()
		exec.Command(start, channel).Run()

		resp, err := http.Get(src)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.Header("Transfer-Encoding", "identity")
		c.Header("Content-Type", "video/mp2t")
		c.Writer.WriteHeaderNow()
		c.Writer.Flush()

		defer func() {
			resp.Body.Close()
			exec.Command(stop).Run()
		}()

		io.Copy(c.Writer, resp.Body)
	})
	return r.Run(":7654")
}
