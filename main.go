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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tarm/serial"
)

var (
	reader    *bufio.Reader
	powerDraw_del = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "power_draw_watts_del",
		Help: "Current delivered power draw in Watts",
	})

	powerDraw_res = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "power_draw_watts_res",
		Help: "Current received power draw in Watts",
	})

	powerTariff1_in = prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "power_meter_tariff1_watthours_in",
			Help: "power meter tariff1 to client reading in Watthours",
		},
		func() float64 {
			return powerTariff1Meter_in
		},
	)

	powerTariff1_out = prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "power_meter_tariff1_watthours_out",
			Help: "power meter tariff1 by client reading in Watthours",
		},
		func() float64 {
			return powerTariff1Meter_out
		},
	)

	powerTariff2_in = prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "power_meter_tariff2_watthours_in",
			Help: "power meter tariff2 to client reading in Watthours",
		},
		func() float64 {
			return powerTariff2Meter_in
		},
	)

	powerTariff2_out = prometheus.NewCounterFunc(
		prometheus.CounterOpts{
			Name: "power_meter_tariff2_watthours_out",
			Help: "power meter tariff2 by client reading in Watthours",
		},
		func() float64 {
			return powerTariff2Meter_out
		},
	)

	powerTariff1Meter_in float64
	powerTariff2Meter_in float64
	powerTariff1Meter_out float64
	powerTariff2Meter_out float64
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(powerDraw_del)
	prometheus.MustRegister(powerTariff1_in)
	prometheus.MustRegister(powerTariff2_in)
	prometheus.MustRegister(powerDraw_res)
	prometheus.MustRegister(powerTariff1_out)
	prometheus.MustRegister(powerTariff2_out)
}

func main() {
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

	fmt.Println("now serving metrics")
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9222", nil))

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
			powerTariff1Meter_in = tmpVal * 1000

		} else if strings.HasPrefix(line, "1-0:2.8.1") {
			tmpVal, err := strconv.ParseFloat(line[10:20], 64)
			if err != nil {
				log.Println(err)
				continue
			}
			powerTariff1Meter_out = tmpVal * 1000

		} else if strings.HasPrefix(line, "1-0:1.8.2") {
			tmpVal, err := strconv.ParseFloat(line[10:20], 64)
			if err != nil {
				log.Println(err)
				continue
			}
			powerTariff2Meter_in = tmpVal * 1000

		} else if strings.HasPrefix(line, "1-0:2.8.2") {
			tmpVal, err := strconv.ParseFloat(line[10:20], 64)
			if err != nil {
				log.Println(err)
				continue
			}
			powerTariff2Meter_out = tmpVal * 1000

		} else if strings.HasPrefix(line, "1-0:1.7.0") {
			tmpVal, err := strconv.ParseFloat(line[10:16], 64)
			if err != nil {
				log.Println(err)
				continue
			}
			powerDraw_del.Set(tmpVal * 1000)

		} else if strings.HasPrefix(line, "1-0:2.7.0") {
			tmpVal, err := strconv.ParseFloat(line[10:16], 64)
			if err != nil {
				log.Println(err)
				continue
			}
			powerDraw_res.Set(tmpVal * 1000)

		}
		if os.Getenv("SERIAL_DEVICE") == "" {
			time.Sleep(200 * time.Millisecond)
		}
	}
}
