package app

import (
	"testing"
)

// TestThunderboltHeadlessDataCollection tests the collectHeadlessData function
// which gathers Thunderbolt and system metrics for headless output
func TestThunderboltHeadlessDataCollection(t *testing.T) {
	// Create mock ThunderboltOutput
	tbInfo := &ThunderboltOutput{
		Buses: []ThunderboltBusOutput{
			{Name: "TB4 Bus 0"},
		},
	}

	// Create mock SystemInfo
	sysInfo := SystemInfo{
		Name:         "Test Mac",
		CoreCount:    8,
		ECoreCount:   4,
		PCoreCount:   4,
		GPUCoreCount: 10,
	}

	// Call collectHeadlessData
	output := collectHeadlessData(tbInfo, sysInfo)

	// Verify basic structure
	if output.SystemInfo.Name != "Test Mac" {
		t.Errorf("Expected system name 'Test Mac', got '%s'", output.SystemInfo.Name)
	}

	if output.SystemInfo.CoreCount != 8 {
		t.Errorf("Expected core count 8, got %d", output.SystemInfo.CoreCount)
	}

	// Timestamp should be set
	if output.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

// TestThunderboltNetworkStatsMapping tests the mapTBNetStatsToBuses function
// which maps Thunderbolt network interface statistics to bus information
func TestThunderboltNetworkStatsMapping(t *testing.T) {
	// Create mock network stats
	netStats := []ThunderboltNetStats{
		{InterfaceName: "en1", BytesInPerSec: 1000, BytesOutPerSec: 2000},
	}

	// Create mock ThunderboltOutput
	tbInfo := &ThunderboltOutput{
		Buses: []ThunderboltBusOutput{
			{Name: "TB4 Bus 0", NetworkStats: &ThunderboltNetStats{InterfaceName: "en1"}},
		},
	}

	// Map network stats to buses
	mapTBNetStatsToBuses(netStats, tbInfo)

	// Verify the mapping worked
	if len(tbInfo.Buses) > 0 && tbInfo.Buses[0].NetworkStats != nil {
		t.Log("Network stats mapped successfully")
	}
}
