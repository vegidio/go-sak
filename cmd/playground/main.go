package main

import (
	"fmt"

	"github.com/vegidio/go-sak/sysinfo"
)

func main() {
	cpu, err := sysinfo.GetCPUInfo()
	if err != nil {
		fmt.Println("Error getting CPU info", err)
		return
	}

	fmt.Println("CPU", cpu)

	memory, err := sysinfo.GetMemoryInfo()
	if err != nil {
		fmt.Println("Error getting memory info", err)
		return
	}

	fmt.Println("Memory", memory)

	gpu, err := sysinfo.GetGPUInfo()
	if err != nil {
		fmt.Println("Error getting GPU info", err)
		return
	}

	fmt.Println("GPU", gpu)
}
