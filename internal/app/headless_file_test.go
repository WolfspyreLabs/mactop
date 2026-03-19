package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestHeadlessSNMPFormat tests the SNMP output format
func TestHeadlessSNMPFormat(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	projectRoot := filepath.Join("..", "..")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", "mactop_test_binary", ".")
	buildCmd.Dir = projectRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, out)
	}
	defer func() {
		if err := os.Remove(filepath.Join(projectRoot, "mactop_test_binary")); err != nil {
			t.Logf("Warning: could not remove test binary: %v", err)
		}
	}()

	cmd := exec.CommandContext(ctx, "./mactop_test_binary", "--headless", "--count", "1", "--format", "snmp")
	cmd.Dir = projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("headless snmp command failed: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if len(output) == 0 {
		t.Fatal("Expected non-empty output from headless SNMP mode")
	}

	// Check for expected SNMP format lines
	expectedMetrics := []string{
		"mactop.cpu.usage=",
		"mactop.gpu.usage=",
		"mactop.memory.used_gb=",
		"mactop.timestamp=",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(output, metric) {
			t.Errorf("Expected output to contain metric '%s', but it was not found", metric)
		}
	}

	// Check that each line is in key=value format
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if !strings.Contains(line, "=") {
			t.Errorf("Line does not contain '=' separator: %s", line)
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			t.Errorf("Line does not have exactly one '=': %s", line)
		}
		if !strings.HasPrefix(parts[0], "mactop.") {
			t.Errorf("Metric name does not start with 'mactop.': %s", line)
		}
	}
}

// TestHeadlessFileOutput tests file output functionality
func TestHeadlessFileOutput(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	projectRoot := filepath.Join("..", "..")
	tmpFile := filepath.Join(os.TempDir(), "mactop_test_file.json")

	// Clean up any existing file
	if err := os.Remove(tmpFile); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: could not remove temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			t.Logf("Warning: could not remove temp file: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", "mactop_test_binary", ".")
	buildCmd.Dir = projectRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, out)
	}
	defer func() {
		if err := os.Remove(filepath.Join(projectRoot, "mactop_test_binary")); err != nil {
			t.Logf("Warning: could not remove test binary: %v", err)
		}
	}()

	// Test file output
	cmd := exec.CommandContext(ctx, "./mactop_test_binary", "--headless", "--count", "1", "--output-file", tmpFile)
	cmd.Dir = projectRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("headless file output command failed: %v\nstderr: %s", err, stderr.String())
	}

	// Check file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("Output file was not created")
	}

	// Check file content
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("Output file is empty")
	}

	// Verify it's valid JSON
	var result []HeadlessOutput
	if err := json.Unmarshal(content, &result); err != nil {
		t.Fatalf("Failed to parse JSON from file: %v\nContent: %s", err, string(content))
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 sample in file, got %d", len(result))
	}
}

// TestHeadlessAppendMode tests append mode functionality with SNMP format
func TestHeadlessAppendMode(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	projectRoot := filepath.Join("..", "..")
	tmpFile := filepath.Join(os.TempDir(), "mactop_test_append_snmp.txt")

	// Clean up any existing file
	if err := os.Remove(tmpFile); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: could not remove temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			t.Logf("Warning: could not remove temp file: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", "mactop_test_binary", ".")
	buildCmd.Dir = projectRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, out)
	}
	defer func() {
		if err := os.Remove(filepath.Join(projectRoot, "mactop_test_binary")); err != nil {
			t.Logf("Warning: could not remove test binary: %v", err)
		}
	}()

	// First write without append (SNMP format)
	cmd1 := exec.CommandContext(ctx, "./mactop_test_binary", "--headless", "--count", "1", "--format", "snmp", "--output-file", tmpFile)
	cmd1.Dir = projectRoot
	if err := cmd1.Run(); err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	content1, _ := os.ReadFile(tmpFile)
	lines1 := strings.Count(string(content1), "\n")

	// Second write with append (SNMP format)
	cmd2 := exec.CommandContext(ctx, "./mactop_test_binary", "--headless", "--count", "1", "--format", "snmp", "--output-file", tmpFile, "--append")
	cmd2.Dir = projectRoot
	if err := cmd2.Run(); err != nil {
		t.Fatalf("Second write with append failed: %v", err)
	}

	content2, _ := os.ReadFile(tmpFile)
	lines2 := strings.Count(string(content2), "\n")

	// With append, we should have more lines than without
	if lines2 <= lines1 {
		t.Errorf("Append mode did not add content: before=%d lines, after=%d lines", lines1, lines2)
	}

	// Verify both samples have timestamp entries
	output := string(content2)
	timestampCount := strings.Count(output, "mactop.timestamp=")
	if timestampCount < 2 {
		t.Errorf("Expected at least 2 timestamp entries with append mode, got %d", timestampCount)
	}
}

// TestHeadlessSNMPFileOutput tests SNMP format with file output
func TestHeadlessSNMPFileOutput(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	projectRoot := filepath.Join("..", "..")
	tmpFile := filepath.Join(os.TempDir(), "mactop_test_snmp.txt")

	// Clean up any existing file
	if err := os.Remove(tmpFile); err != nil && !os.IsNotExist(err) {
		t.Logf("Warning: could not remove temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile); err != nil {
			t.Logf("Warning: could not remove temp file: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", "mactop_test_binary", ".")
	buildCmd.Dir = projectRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, out)
	}
	defer func() {
		if err := os.Remove(filepath.Join(projectRoot, "mactop_test_binary")); err != nil {
			t.Logf("Warning: could not remove test binary: %v", err)
		}
	}()

	// Test SNMP format with file output
	cmd := exec.CommandContext(ctx, "./mactop_test_binary", "--headless", "--count", "1", "--format", "snmp", "--output-file", tmpFile)
	cmd.Dir = projectRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("headless SNMP file output command failed: %v\nstderr: %s", err, stderr.String())
	}

	// Check file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("Output file was not created")
	}

	// Check file content
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("Output file is empty")
	}

	// Verify SNMP format
	output := string(content)
	if !strings.Contains(output, "mactop.cpu.usage=") {
		t.Error("Expected SNMP format to contain mactop.cpu.usage metric")
	}
	if !strings.Contains(output, "mactop.timestamp=") {
		t.Error("Expected SNMP format to contain mactop.timestamp metric")
	}
}
