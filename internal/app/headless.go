package app

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/toon-format/toon-go"
	"gopkg.in/yaml.v3"
)

func safeFloat64At(slice []float64, index int) float64 {
	if index >= 0 && index < len(slice) {
		return slice[index]
	}
	return 0.0
}

// HeadlessProcess represents a single process in headless output
type HeadlessProcess struct {
	PID     int     `json:"pid" yaml:"pid" xml:"PID" toon:"pid"`
	Command string  `json:"command" yaml:"command" xml:"Command" toon:"command"`
	CPU     float64 `json:"cpu_percent" yaml:"cpu_percent" xml:"CPUPercent" toon:"cpu_percent"`
	GPU     float64 `json:"gpu_ms_per_sec" yaml:"gpu_ms_per_sec" xml:"GPUMsPerSec" toon:"gpu_ms_per_sec"`
	Memory  float64 `json:"memory_percent" yaml:"memory_percent" xml:"MemoryPercent" toon:"memory_percent"`
	RSS     int64   `json:"rss_kb" yaml:"rss_kb" xml:"RSSKB" toon:"rss_kb"`
}

// HeadlessNetworkLinks holds link speed info for all network interfaces
type HeadlessNetworkLinks struct {
	Ethernet []HeadlessEthernetLink `json:"ethernet,omitempty" yaml:"ethernet,omitempty" xml:"Ethernet" toon:"ethernet"`
	WiFi     *HeadlessWiFiLink      `json:"wifi,omitempty" yaml:"wifi,omitempty" xml:"WiFi" toon:"wifi"`
}

// HeadlessEthernetLink represents a single Ethernet interface's link info
type HeadlessEthernetLink struct {
	Name           string `json:"name" yaml:"name" xml:"Name" toon:"name"`
	LinkUp         bool   `json:"link_up" yaml:"link_up" xml:"LinkUp" toon:"link_up"`
	SpeedMbps      uint64 `json:"speed_mbps" yaml:"speed_mbps" xml:"SpeedMbps" toon:"speed_mbps"`
	SpeedFormatted string `json:"speed_formatted" yaml:"speed_formatted" xml:"SpeedFormatted" toon:"speed_formatted"`
}

// HeadlessWiFiLink represents Wi-Fi interface link info
type HeadlessWiFiLink struct {
	Interface  string `json:"interface" yaml:"interface" xml:"Interface" toon:"interface"`
	PHYMode    string `json:"phy_mode" yaml:"phy_mode" xml:"PHYMode" toon:"phy_mode"`
	Generation string `json:"generation" yaml:"generation" xml:"Generation" toon:"generation"`
	TxRateMbps int    `json:"tx_rate_mbps" yaml:"tx_rate_mbps" xml:"TxRateMbps" toon:"tx_rate_mbps"`
	Connected  bool   `json:"connected" yaml:"connected" xml:"Connected" toon:"connected"`
}

// HeadlessGPUMetrics holds GPU frequency and utilization
type HeadlessGPUMetrics struct {
	FreqMHz       int     `json:"freq_mhz" yaml:"freq_mhz" xml:"FreqMHz" toon:"freq_mhz"`
	ActivePercent float64 `json:"active_percent" yaml:"active_percent" xml:"ActivePercent" toon:"active_percent"`
}

// HeadlessVolume represents a disk volume's usage
type HeadlessVolume struct {
	Name    string  `json:"name" yaml:"name" xml:"Name" toon:"name"`
	TotalGB float64 `json:"total_gb" yaml:"total_gb" xml:"TotalGB" toon:"total_gb"`
	UsedGB  float64 `json:"used_gb" yaml:"used_gb" xml:"UsedGB" toon:"used_gb"`
	UsedPct float64 `json:"used_percent" yaml:"used_percent" xml:"UsedPercent" toon:"used_percent"`
}

type HeadlessOutput struct {
	Timestamp             string               `json:"timestamp" yaml:"timestamp" xml:"Timestamp" toon:"timestamp"`
	SocMetrics            SocMetrics           `json:"soc_metrics" yaml:"soc_metrics" xml:"SocMetrics" toon:"soc_metrics"`
	Memory                MemoryMetrics        `json:"memory" yaml:"memory" xml:"Memory" toon:"memory"`
	NetDisk               NetDiskMetrics       `json:"net_disk" yaml:"net_disk" xml:"NetDisk" toon:"net_disk"`
	CPUUsage              float64              `json:"cpu_usage" yaml:"cpu_usage" xml:"CPUUsage" toon:"cpu_usage"`
	ECPUUsage             []float64            `json:"ecpu_usage" yaml:"ecpu_usage" xml:"ECPUUsage" toon:"ecpu_usage"`
	PCPUUsage             []float64            `json:"pcpu_usage" yaml:"pcpu_usage" xml:"PCPUUsage" toon:"pcpu_usage"`
	GPUUsage              float64              `json:"gpu_usage" yaml:"gpu_usage" xml:"GPUUsage" toon:"gpu_usage"`
	GPUMetrics            HeadlessGPUMetrics   `json:"gpu_metrics" yaml:"gpu_metrics" xml:"GPUMetrics" toon:"gpu_metrics"`
	TFLOPsFP32            float64              `json:"tflops_fp32" yaml:"tflops_fp32" xml:"TFLOPsFP32" toon:"tflops_fp32"`
	TFLOPsFP16            float64              `json:"tflops_fp16" yaml:"tflops_fp16" xml:"TFLOPsFP16" toon:"tflops_fp16"`
	CoreUsages            []float64            `json:"core_usages" yaml:"core_usages" xml:"CoreUsages" toon:"core_usages"`
	SystemInfo            SystemInfo           `json:"system_info" yaml:"system_info" xml:"SystemInfo" toon:"system_info"`
	ThermalState          string               `json:"thermal_state" yaml:"thermal_state" xml:"ThermalState" toon:"thermal_state"`
	Processes             []HeadlessProcess    `json:"processes,omitempty" yaml:"processes,omitempty" xml:"Processes" toon:"processes"`
	NetworkLinks          HeadlessNetworkLinks `json:"network_links" yaml:"network_links" xml:"NetworkLinks" toon:"network_links"`
	Volumes               []HeadlessVolume     `json:"volumes,omitempty" yaml:"volumes,omitempty" xml:"Volumes" toon:"volumes"`
	ThunderboltInfo       *ThunderboltOutput   `json:"thunderbolt_info" yaml:"thunderbolt_info" xml:"ThunderboltInfo" toon:"thunderbolt_info"`
	TBNetTotalBytesInSec  float64              `json:"tb_net_total_bytes_in_per_sec" yaml:"tb_net_total_bytes_in_per_sec" xml:"TBNetTotalBytesInSec" toon:"tb_net_total_bytes_in_per_sec"`
	TBNetTotalBytesOutSec float64              `json:"tb_net_total_bytes_out_per_sec" yaml:"tb_net_total_bytes_out_per_sec" xml:"TBNetTotalBytesOutSec" toon:"tb_net_total_bytes_out_per_sec"`
	RDMAStatus            RDMAStatus           `json:"rdma_status" yaml:"rdma_status" xml:"RDMAStatus" toon:"rdma_status"`
}

// runHeadless executes headless mode, collecting metrics and outputting them
// in the specified format to stdout or a file.
func runHeadless(count int) {
	if err := initSocMetrics(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize metrics: %v\n", err)
		os.Exit(1)
	}
	defer cleanupSocMetrics()

	startHeadlessPrometheus()

	format := validateFormat(strings.ToLower(headlessFormat))
	tbInfo := performHeadlessWarmup()
	cachedHeadlessSysInfo := getSOCInfo()
	outputFile, keepFileOpen := openHeadlessOutputFile(count)
	if outputFile != nil {
		defer func() {
			if closeErr := outputFile.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Error closing output file: %v\n", closeErr)
			}
		}()
	}

	printHeadlessStart(format, count, outputFile)

	samplesCollected := 0
	if err := processHeadlessSample(format, tbInfo, cachedHeadlessSysInfo, outputFile, keepFileOpen); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
	}
	samplesCollected++

	if count > 0 && samplesCollected >= count {
		printHeadlessEnd(format, count, outputFile, samplesCollected)
		return
	}

	runHeadlessLoop(format, count, tbInfo, cachedHeadlessSysInfo, outputFile, keepFileOpen, &samplesCollected)
}

// validateFormat ensures the format is valid, defaulting to json if unknown.
func validateFormat(format string) string {
	switch format {
	case "json", "yaml", "xml", "toon", "csv", "snmp":
		return format
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s. Defaulting to json.\n", format)
		return "json"
	}
}

// openHeadlessOutputFile opens the output file if specified and needed.
func openHeadlessOutputFile(count int) (*os.File, bool) {
	keepFileOpen := headlessAppend || count > 0
	if headlessOutputFile == "" || !keepFileOpen {
		return nil, keepFileOpen
	}

	flags := os.O_CREATE | os.O_WRONLY
	if headlessAppend {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	outputFile, err := os.OpenFile(headlessOutputFile, flags, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open output file: %v\n", err)
		os.Exit(1)
	}

	return outputFile, keepFileOpen
}

// runHeadlessLoop runs the main collection loop for headless mode.
func runHeadlessLoop(format string, count int, tbInfo *ThunderboltOutput, sysInfo SystemInfo, outputFile *os.File, keepFileOpen bool, samplesCollected *int) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(updateInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			printHeadlessEnd(format, count, outputFile, *samplesCollected)
			return
		case <-ticker.C:
			printHeadlessSeparator(format, count, *samplesCollected, outputFile)

			if err := processHeadlessSample(format, tbInfo, sysInfo, outputFile, keepFileOpen); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			}

			(*samplesCollected)++
			if count > 0 && *samplesCollected >= count {
				printHeadlessEnd(format, count, outputFile, *samplesCollected)
				return
			}
		}
	}
}

// printHeadlessStart writes the opening format for headless output.
// For JSON with count > 0, writes opening bracket. For XML, writes root element.
// For CSV, writes the header row.
func printHeadlessStart(format string, count int, outputFile *os.File) {
	writer := getHeadlessWriter(outputFile)
	if count > 0 {
		switch format {
		case "json":
			// Check if file exists and has content when appending
			if headlessAppend && outputFile != nil {
				// Seek to end to check current position
				pos, err := outputFile.Seek(0, io.SeekEnd)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error seeking output file: %v\n", err)
				}
				if pos > 0 {
					// File has content, need to write comma before next object
					if _, err := fmt.Fprint(writer, ","); err != nil {
						fmt.Fprintf(os.Stderr, "Error writing to output file: %v\n", err)
					}
					return
				}
			}
			if _, err := fmt.Fprint(writer, "["); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to output file: %v\n", err)
			}
		case "xml":
			if _, err := fmt.Fprint(writer, "<MactopOutputList>"); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to output file: %v\n", err)
			}
		case "csv":
			printCSVHeader(writer)
		}
	} else {
		switch format {
		case "xml":
			// XML always needs a root element, even in infinite mode
			if _, err := fmt.Fprint(writer, "<MactopOutputList>"); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to output file: %v\n", err)
			}
		case "csv":
			printCSVHeader(writer)
		}
	}
}

// getHeadlessWriter returns the appropriate writer for headless output.
// If outputFile is provided, it returns the file; otherwise returns stdout.
func getHeadlessWriter(outputFile *os.File) io.Writer {
	if outputFile != nil {
		return outputFile
	}
	return os.Stdout
}

// printCSVHeader writes the CSV header line to the provided writer.
func printCSVHeader(writer io.Writer) {
	headers := []string{
		"Timestamp",
		"System_Name", "Core_Count", "E_Core_Count", "P_Core_Count", "GPU_Core_Count",
		"CPU_Usage", "ECPU_Freq_MHz", "ECPU_Active", "PCPU_Freq_MHz", "PCPU_Active", "GPU_Usage",
		"GPU_Freq_MHz", "GPU_Active_Percent",
		"Mem_Used", "Mem_Total", "Swap_Used",
		"Disk_Read_KB", "Disk_Write_KB",
		"Net_In_Bytes", "Net_Out_Bytes",
		"TB_Net_In_Bytes", "TB_Net_Out_Bytes",
		"Total_Power", "System_Power",
		"CPU_Temp", "GPU_Temp", "Thermal_State",
		"RDMA_Available", "RDMA_Status", "RDMA_Device_Count",
	}

	// Add dynamic core headers
	sysInfo := getSOCInfo()
	for i := 0; i < sysInfo.CoreCount; i++ {
		headers = append(headers, fmt.Sprintf("Core_%d", i))
	}

	// Add JSON blob headers for complex nested data
	headers = append(headers, "Thunderbolt_Info_JSON", "Processes_JSON", "Network_Links_JSON", "Volumes_JSON")

	// Print CSV header line
	if _, err := fmt.Fprintln(writer, strings.Join(headers, ",")); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing CSV header: %v\n", err)
	}
}

// printHeadlessEnd writes the closing format for headless output.
// For JSON with count > 0, writes closing bracket. For XML, writes closing root element.
func printHeadlessEnd(format string, count int, outputFile *os.File, samplesCollected int) {
	writer := getHeadlessWriter(outputFile)
	if count > 0 {
		switch format {
		case "json":
			// Only close the array if we're not in append mode or this is the final sample
			if !headlessAppend || samplesCollected >= count {
				if _, err := fmt.Fprintln(writer, "]"); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing to output file: %v\n", err)
				}
			}
		case "xml":
			if _, err := fmt.Fprintln(writer, "</MactopOutputList>"); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to output file: %v\n", err)
			}
		}
	} else if format == "xml" {
		if _, err := fmt.Fprintln(writer, "</MactopOutputList>"); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to output file: %v\n", err)
		}
	}
}

// printHeadlessSeparator writes separators between samples in headless output.
// For JSON, writes comma. For YAML, writes document separator (---).
func printHeadlessSeparator(format string, count int, samplesCollected int, outputFile *os.File) {
	writer := getHeadlessWriter(outputFile)
	if samplesCollected > 0 && count > 0 {
		switch format {
		case "json":
			if _, err := fmt.Fprint(writer, ","); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing separator: %v\n", err)
			}
		case "yaml":
			if _, err := fmt.Fprintln(writer, "---"); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing separator: %v\n", err)
			}
		}
	} else if format == "yaml" {
		// Even for infinite stream, YAML docs are best separated by ---
		if _, err := fmt.Fprintln(writer, "---"); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing separator: %v\n", err)
		}
	}
}

func startHeadlessPrometheus() {
	if prometheusPort != "" {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe(prometheusPort, nil); err != nil {
				fmt.Fprintf(os.Stderr, "Prometheus server error: %v\n", err)
			}
		}()
	}
}

func performHeadlessWarmup() *ThunderboltOutput {
	if _, err := GetCPUPercentages(); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting CPU percentages: %v\n", err)
	}
	getNetDiskMetrics()
	GetThunderboltNetStats()

	startInit := time.Now()
	tbInfo, _ := GetFormattedThunderboltInfo()
	initDuration := time.Since(startInit)

	initialDelay := time.Duration(updateInterval)*time.Millisecond - initDuration
	if initialDelay > 0 {
		time.Sleep(initialDelay)
	}
	return tbInfo
}

// processHeadlessSample collects a single sample of metrics and outputs it
// in the specified format. If outputFile is provided, writes to file; otherwise stdout.
func processHeadlessSample(format string, tbInfo *ThunderboltOutput, sysInfo SystemInfo, outputFile *os.File, keepFileOpen bool) error {
	output := collectHeadlessData(tbInfo, sysInfo)
	writer, closeWriter := getHeadlessWriterWithClose(outputFile, keepFileOpen, format)
	defer closeWriter()

	switch format {
	case "csv":
		return writeCSVOutput(writer, output)
	case "snmp":
		return writeSNMPOutput(writer, output)
	case "json", "yaml", "xml", "toon":
		return writeStructuredOutput(writer, format, output)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// getHeadlessWriterWithClose returns a writer and a close function for headless output.
func getHeadlessWriterWithClose(outputFile *os.File, keepFileOpen bool, format string) (io.Writer, func()) {
	if !keepFileOpen && headlessOutputFile != "" {
		// Open file for this sample only (overwrite mode, infinite)
		tempFile, err := os.OpenFile(headlessOutputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open output file: %v\n", err)
			return os.Stdout, func() {}
		}

		// For CSV format, write the header each time in overwrite mode
		if format == "csv" {
			printCSVHeader(tempFile)
		}

		return tempFile, func() {
			if closeErr := tempFile.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Error closing temp file: %v\n", closeErr)
			}
		}
	}

	return getHeadlessWriter(outputFile), func() {}
}

// writeCSVOutput writes metrics in CSV format.
func writeCSVOutput(writer io.Writer, output HeadlessOutput) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	var record []string
	record = append(record,
		output.Timestamp,
		output.SystemInfo.Name,
		fmt.Sprintf("%d", output.SystemInfo.CoreCount),
		fmt.Sprintf("%d", output.SystemInfo.ECoreCount),
		fmt.Sprintf("%d", output.SystemInfo.PCoreCount),
		fmt.Sprintf("%d", output.SystemInfo.GPUCoreCount),
		fmt.Sprintf("%.2f", output.CPUUsage),
		fmt.Sprintf("%.2f", safeFloat64At(output.ECPUUsage, 0)),
		fmt.Sprintf("%.2f", safeFloat64At(output.ECPUUsage, 1)),
		fmt.Sprintf("%.2f", safeFloat64At(output.PCPUUsage, 0)),
		fmt.Sprintf("%.2f", safeFloat64At(output.PCPUUsage, 1)),
		fmt.Sprintf("%.2f", output.GPUUsage),
		fmt.Sprintf("%d", output.GPUMetrics.FreqMHz),
		fmt.Sprintf("%.2f", output.GPUMetrics.ActivePercent),
		fmt.Sprintf("%d", output.Memory.Used),
		fmt.Sprintf("%d", output.Memory.Total),
		fmt.Sprintf("%d", output.Memory.SwapUsed),
		fmt.Sprintf("%.2f", output.NetDisk.ReadKBytesPerSec),
		fmt.Sprintf("%.2f", output.NetDisk.WriteKBytesPerSec),
		fmt.Sprintf("%.2f", output.NetDisk.InBytesPerSec),
		fmt.Sprintf("%.2f", output.NetDisk.OutBytesPerSec),
		fmt.Sprintf("%.2f", output.TBNetTotalBytesInSec),
		fmt.Sprintf("%.2f", output.TBNetTotalBytesOutSec),
		fmt.Sprintf("%.2f", output.SocMetrics.TotalPower),
		fmt.Sprintf("%.2f", output.SocMetrics.SystemPower),
		fmt.Sprintf("%.2f", output.SocMetrics.CPUTemp),
		fmt.Sprintf("%.2f", output.SocMetrics.GPUTemp),
		output.ThermalState,
		fmt.Sprintf("%t", output.RDMAStatus.Available),
		output.RDMAStatus.Status,
		fmt.Sprintf("%d", len(output.RDMAStatus.Devices)),
	)

	for i := 0; i < output.SystemInfo.CoreCount; i++ {
		val := 0.0
		if i < len(output.CoreUsages) {
			val = output.CoreUsages[i]
		}
		record = append(record, fmt.Sprintf("%.2f", val))
	}

	tbJSON, _ := json.Marshal(output.ThunderboltInfo)
	procsJSON, _ := json.Marshal(output.Processes)
	linksJSON, _ := json.Marshal(output.NetworkLinks)
	volsJSON, _ := json.Marshal(output.Volumes)
	record = append(record, string(tbJSON), string(procsJSON), string(linksJSON), string(volsJSON))

	if err := csvWriter.Write(record); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing CSV record: %v\n", err)
	}
	return nil
}

// writeSNMPOutput writes metrics in SNMP key=value format.
func writeSNMPOutput(writer io.Writer, output HeadlessOutput) error {
	lines := []string{
		fmt.Sprintf("mactop.cpu.usage=%.2f", output.CPUUsage),
		fmt.Sprintf("mactop.cpu.ecpu_freq_mhz=%.0f", safeFloat64At(output.ECPUUsage, 0)),
		fmt.Sprintf("mactop.cpu.pcpu_freq_mhz=%.0f", safeFloat64At(output.PCPUUsage, 0)),
		fmt.Sprintf("mactop.gpu.usage=%.2f", output.GPUUsage),
		fmt.Sprintf("mactop.gpu.freq_mhz=%d", output.GPUMetrics.FreqMHz),
		fmt.Sprintf("mactop.memory.used_gb=%.2f", float64(output.Memory.Used)/1024/1024/1024),
		fmt.Sprintf("mactop.memory.total_gb=%.2f", float64(output.Memory.Total)/1024/1024/1024),
		fmt.Sprintf("mactop.memory.swap_used_gb=%.2f", float64(output.Memory.SwapUsed)/1024/1024/1024),
		fmt.Sprintf("mactop.network.in_bytes_per_sec=%.2f", output.NetDisk.InBytesPerSec),
		fmt.Sprintf("mactop.network.out_bytes_per_sec=%.2f", output.NetDisk.OutBytesPerSec),
		fmt.Sprintf("mactop.disk.read_kbytes_per_sec=%.2f", output.NetDisk.ReadKBytesPerSec),
		fmt.Sprintf("mactop.disk.write_kbytes_per_sec=%.2f", output.NetDisk.WriteKBytesPerSec),
		fmt.Sprintf("mactop.power.total_watts=%.2f", output.SocMetrics.TotalPower),
		fmt.Sprintf("mactop.power.system_watts=%.2f", output.SocMetrics.SystemPower),
		fmt.Sprintf("mactop.temp.cpu_celsius=%.2f", output.SocMetrics.CPUTemp),
		fmt.Sprintf("mactop.temp.gpu_celsius=%.2f", output.SocMetrics.GPUTemp),
		fmt.Sprintf("mactop.thermal.state=%s", output.ThermalState),
		fmt.Sprintf("mactop.rdma.available=%t", output.RDMAStatus.Available),
		fmt.Sprintf("mactop.timestamp=%s", output.Timestamp),
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(writer, line); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing SNMP line: %v\n", err)
		}
	}
	return nil
}

// writeStructuredOutput writes metrics in structured formats (JSON, YAML, XML, toon).
func writeStructuredOutput(writer io.Writer, format string, output HeadlessOutput) error {
	var data []byte
	var err error

	switch format {
	case "json":
		if headlessPretty {
			data, err = json.MarshalIndent(output, "", "  ")
		} else {
			data, err = json.Marshal(output)
		}
	case "yaml":
		data, err = yaml.Marshal(output)
	case "xml":
		if headlessPretty {
			data, err = xml.MarshalIndent(output, "", "  ")
		} else {
			data, err = xml.Marshal(output)
		}
	case "toon":
		data, err = toon.Marshal(output)
	}

	if err != nil {
		return err
	}

	if _, err := fmt.Fprintln(writer, string(data)); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
	}
	return nil
}

// headless link info cache (refreshed every 5s like TUI)
var (
	headlessLinkInfoMutex      sync.RWMutex
	headlessEthernetLinkInfo   []EthernetLinkInfo
	headlessWiFiLinkInfo       *WiFiLinkInfo
	headlessLinkInfoLastUpdate time.Time
)

func getHeadlessNetworkLinks() HeadlessNetworkLinks {
	headlessLinkInfoMutex.RLock()
	needsRefresh := time.Since(headlessLinkInfoLastUpdate) >= 5*time.Second
	headlessLinkInfoMutex.RUnlock()

	if needsRefresh {
		headlessLinkInfoMutex.Lock()
		if time.Since(headlessLinkInfoLastUpdate) >= 5*time.Second {
			headlessEthernetLinkInfo = GetEthernetLinkInfo()
			headlessWiFiLinkInfo = GetWiFiLinkInfo()
			headlessLinkInfoLastUpdate = time.Now()
		}
		headlessLinkInfoMutex.Unlock()
	}

	headlessLinkInfoMutex.RLock()
	defer headlessLinkInfoMutex.RUnlock()

	var links HeadlessNetworkLinks
	for _, eth := range headlessEthernetLinkInfo {
		links.Ethernet = append(links.Ethernet, HeadlessEthernetLink{
			Name:           eth.Name,
			LinkUp:         eth.LinkUp,
			SpeedMbps:      eth.LinkSpeedMbps,
			SpeedFormatted: FormatLinkSpeed(eth.LinkSpeedMbps),
		})
	}
	if headlessWiFiLinkInfo != nil {
		links.WiFi = &HeadlessWiFiLink{
			Interface:  headlessWiFiLinkInfo.InterfaceName,
			PHYMode:    headlessWiFiLinkInfo.PHYMode,
			Generation: headlessWiFiLinkInfo.WiFiGeneration,
			TxRateMbps: headlessWiFiLinkInfo.TxRateMbps,
			Connected:  headlessWiFiLinkInfo.IsConnected,
		}
	}
	return links
}

func collectHeadlessData(tbInfo *ThunderboltOutput, sysInfo SystemInfo) HeadlessOutput {
	m := sampleSocMetrics(updateInterval)
	mem := getMemoryMetrics()
	netDisk := getNetDiskMetrics()

	var cpuUsage float64
	percentages, err := GetCPUPercentages()
	if err == nil && len(percentages) > 0 {
		var total float64
		for _, p := range percentages {
			total += p
		}
		cpuUsage = total / float64(len(percentages))
	}

	thermalStr, _ := getThermalStateString()

	componentSum := m.TotalPower
	totalPower := m.SystemPower

	if totalPower < componentSum {
		totalPower = componentSum
	}

	residualSystem := totalPower - componentSum

	m.SystemPower = residualSystem
	m.TotalPower = totalPower

	tbNetStats := GetThunderboltNetStats()
	var tbNetTotalIn, tbNetTotalOut float64
	for _, stat := range tbNetStats {
		tbNetTotalIn += stat.BytesInPerSec
		tbNetTotalOut += stat.BytesOutPerSec
	}

	mapTBNetStatsToBuses(tbNetStats, tbInfo)

	// Get RDMA status and map devices to TB buses
	rdmaStatus := CheckRDMAAvailable()
	mapRDMADevicesToBuses(rdmaStatus.Devices, tbInfo)

	// Calculate TFLOPs
	var fp32TFLOPs, fp16TFLOPs float64
	maxGPUFreq := GetMaxGPUFrequency()
	if maxGPUFreq > 0 && sysInfo.GPUCoreCount > 0 {
		fp32TFLOPs = float64(sysInfo.GPUCoreCount) * float64(maxGPUFreq) * 0.000256
		fp16TFLOPs = fp32TFLOPs * 2
	}

	// Collect per-process metrics (top 20 by CPU, includes GPU time)
	var headlessProcesses []HeadlessProcess
	if procs, err := getProcessList(m.GPUActive); err == nil {
		limit := min(len(procs), 20)
		for _, p := range procs[:limit] {
			headlessProcesses = append(headlessProcesses, HeadlessProcess{
				PID:     p.PID,
				Command: p.Command,
				CPU:     p.CPU,
				GPU:     p.GPU,
				Memory:  p.Memory,
				RSS:     p.RSS,
			})
		}
	}

	// Collect network link speed info
	networkLinks := getHeadlessNetworkLinks()

	// Collect disk volume info
	var headlessVolumes []HeadlessVolume
	for _, v := range getVolumes() {
		headlessVolumes = append(headlessVolumes, HeadlessVolume{
			Name:    v.Name,
			TotalGB: v.Total,
			UsedGB:  v.Used,
			UsedPct: v.UsedPct,
		})
	}

	return HeadlessOutput{
		Timestamp:             time.Now().Format(time.RFC3339),
		SocMetrics:            m,
		Memory:                mem,
		NetDisk:               netDisk,
		CPUUsage:              cpuUsage,
		ECPUUsage:             []float64{float64(m.EClusterFreqMHz), m.EClusterActive},
		PCPUUsage:             []float64{float64(m.PClusterFreqMHz), m.PClusterActive},
		GPUUsage:              m.GPUActive,
		GPUMetrics:            HeadlessGPUMetrics{FreqMHz: int(m.GPUFreqMHz), ActivePercent: m.GPUActive},
		TFLOPsFP32:            fp32TFLOPs,
		TFLOPsFP16:            fp16TFLOPs,
		CoreUsages:            percentages,
		SystemInfo:            sysInfo,
		Processes:             headlessProcesses,
		NetworkLinks:          networkLinks,
		Volumes:               headlessVolumes,
		ThunderboltInfo:       tbInfo,
		TBNetTotalBytesInSec:  tbNetTotalIn,
		TBNetTotalBytesOutSec: tbNetTotalOut,
		RDMAStatus:            rdmaStatus,
		ThermalState:          thermalStr,
	}
}

func mapTBNetStatsToBuses(tbNetStats []ThunderboltNetStats, tbInfo *ThunderboltOutput) {
	// Sort and assign TB Net Stats to Buses
	var enStats []ThunderboltNetStats
	for _, stat := range tbNetStats {
		if strings.HasPrefix(stat.InterfaceName, "en") {
			enStats = append(enStats, stat)
		}
	}

	// Sort en stats by interface number (en2, en3, ...)
	sort.Slice(enStats, func(i, j int) bool {
		// Extract number from enX
		getNu := func(s string) int {
			numStr := strings.TrimPrefix(s, "en")
			n, _ := strconv.Atoi(numStr)
			return n
		}
		return getNu(enStats[i].InterfaceName) < getNu(enStats[j].InterfaceName)
	})

	// Assign to buses based on sorted order (Ordinal Mapping)
	// We sort buses by ID (0, 1, 2...) and map them to sorted interfaces (en2, en3, en4...)
	if tbInfo != nil && len(tbInfo.Buses) > 0 {
		type busIndex struct {
			originalIndex int
			id            int
		}
		var sortedBuses []busIndex

		for i, bus := range tbInfo.Buses {
			// Format is typically "TB4 Bus 5" or "TB4 @ TB3 Bus 3"
			// The bus number is always the last element
			parts := strings.Fields(bus.Name)
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				if busID, err := strconv.Atoi(lastPart); err == nil {
					sortedBuses = append(sortedBuses, busIndex{i, busID})
				}
			}
		}

		// Sort buses by ID
		sort.Slice(sortedBuses, func(i, j int) bool {
			return sortedBuses[i].id < sortedBuses[j].id
		})

		// Assign stats ordinally
		for i := 0; i < len(enStats) && i < len(sortedBuses); i++ {
			// Get the target bus using the original index from our sorted list
			busIdx := sortedBuses[i].originalIndex
			if busIdx >= 0 && busIdx < len(tbInfo.Buses) {
				stat := enStats[i] // Copy for safe pointer reference
				tbInfo.Buses[busIdx].NetworkStats = &stat
			}
		}
	}
}

// mapRDMADevicesToBuses associates RDMA devices with their corresponding TB buses
// by matching the RDMA device interface (e.g., "en2") with the bus NetworkStats interface
func mapRDMADevicesToBuses(rdmaDevices []RDMADevice, tbInfo *ThunderboltOutput) {
	if tbInfo == nil || len(rdmaDevices) == 0 {
		return
	}

	// Build a map of interface name to RDMA device for quick lookup
	rdmaByInterface := make(map[string]*RDMADevice)
	for i := range rdmaDevices {
		if rdmaDevices[i].Interface != "" {
			rdmaByInterface[rdmaDevices[i].Interface] = &rdmaDevices[i]
		}
	}

	// Match RDMA devices to buses based on NetworkStats interface name
	for i := range tbInfo.Buses {
		bus := &tbInfo.Buses[i]
		if bus.NetworkStats != nil && bus.NetworkStats.InterfaceName != "" {
			if rdmaDev, ok := rdmaByInterface[bus.NetworkStats.InterfaceName]; ok {
				bus.RDMADevice = rdmaDev
			}
		}
	}
}
