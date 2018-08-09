package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"flag"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tarm/serial"
)

var (
	reader    *bufio.Reader

	powerDraw = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "power_draw_watts",
			Help: "Current power draw in Watts",
		},
		[]string{"direction"},
	)

	powerTariff = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Help: "power meter reading in Watthours",
			Name: "power_meter_watthours",
		},
		[]string{"metering", "direction"},
	)

)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(powerDraw)
	prometheus.MustRegister(powerTariff)
}

func main() {

	var (
		listenAddress = flag.String("web.listen-address", ":9357", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	)

	flag.Parse()

	// if SERIAL_DEVICE variable is set use it, otherwise read test files
	if os.Getenv("SERIAL_DEVICE") != "" {
		log.Println("gonna use serial device")
		config := &serial.Config{Name: os.Getenv("SERIAL_DEVICE"), Baud: 115200}

		usb, err := serial.OpenPort(config)

		if err != nil {
			log.Fatalf("Could not open serial interface: %s", err)
			return
		}

		reader = bufio.NewReader(usb)

	} else {
		log.Println("gonna use some files")
		file, err := os.Open("examples/fulllist.txt")

		if err != nil {
			log.Fatalln(err)
			return
		}

		defer file.Close()
		reader = bufio.NewReader(file)

	}

	go listener(reader)

	// sleeping 10 seconds to prevent uninitialized scrapes
	time.Sleep(10 * time.Second)

	// Start Prometheus listener
	log.Println("now serving metrics")
	http.Handle(*metricsPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(*listenAddress, nil))

}

func listener(source io.Reader) {
	var line string

	for {
		rawLine, err := reader.ReadBytes('\x0a')

		if err != nil {
			log.Fatalln(err)
			return
		}

		line = string(rawLine[:])

		if strings.HasPrefix(line, "1-0:1.8.1") {
			tmpVal, err := strconv.ParseFloat(line[10:20], 64)

			if err != nil {
				log.Println(err)
				continue
			}

			powerTariff.WithLabelValues("tarrif1","delivered").Set(tmpVal * 1000)

		} else if strings.HasPrefix(line, "1-0:2.8.1") {
			tmpVal, err := strconv.ParseFloat(line[10:20], 64)

			if err != nil {
				log.Println(err)
				continue
			}

			powerTariff.WithLabelValues("tarrif1", "received").Set(tmpVal * 1000)

		} else if strings.HasPrefix(line, "1-0:1.8.2") {
			tmpVal, err := strconv.ParseFloat(line[10:20], 64)

			if err != nil {
				log.Println(err)
				continue
			}

			powerTariff.WithLabelValues("tarrif2", "delivered").Set(tmpVal * 1000)

		} else if strings.HasPrefix(line, "1-0:2.8.2") {
			tmpVal, err := strconv.ParseFloat(line[10:20], 64)

			if err != nil {
				log.Println(err)
				continue
			}

			powerTariff.WithLabelValues("tarrif2", "received").Set(tmpVal * 1000)

		} else if strings.HasPrefix(line, "1-0:1.7.0") {
			tmpVal, err := strconv.ParseFloat(line[10:16], 64)

			if err != nil {
				log.Println(err)
				continue
			}

			powerDraw.WithLabelValues("delivered").Set(tmpVal * 1000)

		} else if strings.HasPrefix(line, "1-0:2.7.0") {
			tmpVal, err := strconv.ParseFloat(line[10:16], 64)

			if err != nil {
				log.Println(err)
				continue
			}

			powerDraw.WithLabelValues("received").Set(tmpVal * 1000)

		}

		if os.Getenv("SERIAL_DEVICE") == "" {
			time.Sleep(200 * time.Millisecond)
		}

	}

}
