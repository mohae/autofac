package main

import (
	"strings"

	"github.com/mohae/systeminfo"
	czap "github.com/mohae/zap"
	"github.com/uber-go/zap"
)

// WriteSystemInfo gathers information about the local system and writes
// it out to the data destination.  If an error occurs, the error will be
// written to both the error log and the data destination.
func WriteSystemInfo() {
	// If the node information is to be written do it now
	var s systeminfo.System
	err := s.Get()
	if err != nil {
		log.Warn(
			err.Error(),
			zap.String("op", "get system information"),
		)
		data.Warn(
			"systeminfo",
			czap.String("error", err.Error()),
		)
		return
	}
	// In multi-processor systems, only the information from the first
	// processor is captured; it is assumed that all processors on the
	// system are the same.
	infs := strings.Join(s.NetInfs, " ")
	data.Warn(
		"systeminfo",
		czap.String("kernel", s.KernelOS),
		czap.String("version", s.KernelVersion),
		czap.String("arch", s.KernelArch),
		czap.String("type", s.KernelType),
		czap.String("compile_date", s.KernelCompileDate),
		czap.String("os", s.OSName),
		czap.String("id", s.OSID),
		czap.String("version_id", s.OSVersion),
		czap.Int64("mem_total", s.MemTotal),
		czap.Int64("swap_total", s.SwapTotal),
		czap.String("net_infs", infs),
		czap.Int("processors", len(s.Chips)),
		czap.String("cpu_vendor", s.Chips[0].VendorID),
		czap.String("cpu_family", s.Chips[0].CPUFamily),
		czap.String("model", s.Chips[0].Model),
		czap.String("model_name", s.Chips[0].ModelName),
		czap.String("stepping", s.Chips[0].Stepping),
		czap.String("microcode", s.Chips[0].Microcode),
		czap.Float64("cpu_mhz", float64(s.Chips[0].CPUMHz)),
		czap.String("cache_size", s.Chips[0].CacheSize),
		czap.Int("cpu_cores", int(s.Chips[0].CPUCores)),
	)
}
