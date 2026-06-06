package main

import (
	"fmt"
	"html/template"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

var startTime = time.Now()

type Item struct {
	Name  string
	Value string
}

type Page struct {
	Title  string
	Groups map[string][]Item
}

func main() {
	e := echo.New()

	t := template.Must(template.ParseFiles("templates/index.html"))

	e.GET("/", func(c echo.Context) error {

		page := buildPage()

		return t.Execute(c.Response(), page)
	})

	e.GET("/readyz", func(c echo.Context) error {
    	return c.NoContent(200)
	})

	e.GET("/livez", func(c echo.Context) error {
    	return c.NoContent(200)
	})

	e.Logger.Fatal(e.Start(":8080"))
}

func buildPage() Page {

	page := Page{
		Title:  "Container Infomation",
		Groups: map[string][]Item{},
	}

	hostname, _ := os.Hostname()

	hi, _ := host.Info()

	vm, _ := mem.VirtualMemory()

	du, _ := disk.Usage("/")

	cpuInfo, _ := cpu.Info()

	cpuUsage, _ := cpu.Percent(time.Second, false)

	cpuModel := "Unknown"

	if len(cpuInfo) > 0 {
		cpuModel = cpuInfo[0].ModelName
	}

	page.Groups["System"] = []Item{
		{"Hostname", hostname},
		{"OS", runtime.GOOS},
		{"Arch", runtime.GOARCH},
		{"Platform", hi.Platform},
		{"Platform Version", hi.PlatformVersion},
		{"Kernel", hi.KernelVersion},
	}

	page.Groups["Runtime"] = []Item{
		{"Go Version", runtime.Version()},
		{"PID", fmt.Sprint(os.Getpid())},
		{"CPU Count", fmt.Sprint(runtime.NumCPU())},
		{"Goroutines", fmt.Sprint(runtime.NumGoroutine())},
		{"Uptime", time.Since(startTime).Truncate(time.Second).String()},
	}

	page.Groups["Resources"] = []Item{
		{"CPU Usage", fmt.Sprintf("%.1f%%", cpuUsage[0])},
		{"Memory Total", formatBytes(vm.Total)},
		{"Memory Used", formatBytes(vm.Used)},
		{"Memory Usage", fmt.Sprintf("%.1f%%", vm.UsedPercent)},
		{"Disk Total", formatBytes(du.Total)},
		{"Disk Used", formatBytes(du.Used)},
		{"Disk Usage", fmt.Sprintf("%.1f%%", du.UsedPercent)},
		{"CPU Model", cpuModel},
	}

	page.Groups["Network"] = collectNetwork()

	if isKubernetes() {
		page.Groups["Kubernetes"] = collectKubernetes()
	}

	return page
}

func collectNetwork() []Item {

	var items []Item

	ifaces, _ := net.Interfaces()

	for _, iface := range ifaces {

		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		if iface.HardwareAddr.String() != "" {
			items = append(items, Item{
				fmt.Sprintf("%s MAC", iface.Name),
				iface.HardwareAddr.String(),
			})
		}

		addrs, _ := iface.Addrs()

		for _, addr := range addrs {

			ipnet, ok := addr.(*net.IPNet)

			if !ok {
				continue
			}

			if ipnet.IP.To4() == nil {
				continue
			}

			items = append(items, Item{
				fmt.Sprintf("%s IPv4", iface.Name),
				ipnet.IP.String(),
			})
		}
	}

	return items
}

func isKubernetes() bool {

	_, err := os.Stat(
		"/var/run/secrets/kubernetes.io/serviceaccount/token",
	)

	return err == nil
}

func collectKubernetes() []Item {

	var items []Item

	hostname, _ := os.Hostname()

	items = append(items, Item{
		"Pod Name",
		hostname,
	})

	nsBytes, err := os.ReadFile(
		"/var/run/secrets/kubernetes.io/serviceaccount/namespace",
	)

	if err == nil {

		items = append(items, Item{
			"Namespace",
			strings.TrimSpace(string(nsBytes)),
		})
	}

	if _, err := os.Stat(
		"/var/run/secrets/kubernetes.io/serviceaccount/token",
	); err == nil {

		items = append(items, Item{
			"ServiceAccount",
			"Mounted",
		})
	}

	clusterIPs, err := net.LookupHost(
		"kubernetes.default.svc",
	)

	if err == nil {

		items = append(items, Item{
			"Cluster IP",
			strings.Join(clusterIPs, ", "),
		})
	}

	return items
}

func formatBytes(v uint64) string {

	const unit = 1024

	if v < unit {
		return fmt.Sprintf("%d B", v)
	}

	div, exp := uint64(unit), 0

	for n := v / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf(
		"%.2f %cB",
		float64(v)/float64(div),
		"KMGTPE"[exp],
	)
}
