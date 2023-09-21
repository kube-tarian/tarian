package nodeagent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// ContainerIDLength is the standard length of the Container ID
	ContainerIDLength = 64

	// BpfContainerIDLength Minimum 31 chars to assume it is a Container ID
	// in case it was truncated
	BpfContainerIDLength = 31

	// DockerIDLength is the length of a Docker container ID.
	DockerIDLength = 128

	// HostProcDir is the path to the host's /proc directory.
	HostProcDir = "/host/proc"
)

// procsContainerID retrieves the container ID associated with a given process ID (PID).
// It reads the cgroup information for the process and extracts the container ID.
//
// Parameters:
//   - pid: The process ID for which to retrieve the container ID.
//
// Returns:
//   - string: The container ID as a string.
//   - error: An error if reading the cgroup information or extracting the container ID fails.
func procsContainerID(pid uint32) (string, error) {
	pidstr := fmt.Sprint(pid)

	// Read the cgroup information for the process.
	cgroups, err := os.ReadFile(filepath.Join(HostProcDir, pidstr, "cgroup"))
	if err != nil {
		return "", err
	}

	// Find the Docker container ID from the cgroup information.
	containerID := findDockerIDFromCgroup(string(cgroups))
	return containerID, nil
}

// findDockerIDFromCgroup searches for the Docker container ID within a given cgroup string.
// It iterates through cgroup paths and looks for identifiers like "pod," "docker," or "libpod"
// to identify the container.
//
// Parameters:
//   - cgroups: The cgroup information as a string.
//
// Returns:
//   - string: The Docker container ID as a string, or an empty string if not found.
func findDockerIDFromCgroup(cgroups string) string {
	cgrpPaths := strings.Split(cgroups, "\n")
	for _, s := range cgrpPaths {
		if strings.Contains(s, "pod") || strings.Contains(s, "docker") ||
			strings.Contains(s, "libpod") {
			// Get the container ID and the offset.
			container := lookupContainerID(s)
			if container != "" {
				return container
			}
		}
	}
	return ""
}

// procsContainerIDOffset extracts the container ID and its offset within a cgroup subdirectory.
// It is used when the cgroup driver is cgroupfs.
//
// Parameters:
//   - subdir: The cgroup subdirectory containing the container information.
//
// Returns:
//   - string: The container ID as a string.
//   - int: The offset of the container ID within the subdirectory.
func procsContainerIDOffset(subdir string) (string, int) {
	// If the cgroup subdir contains ":", it means that we are dealing with
	// Linux.CgroupPath where the cgroup driver is cgroupfs.
	// In this case, split the name and take the last one.
	p := strings.LastIndex(subdir, ":") + 1
	fields := strings.Split(subdir, ":")
	idStr := fields[len(fields)-1]

	off := strings.LastIndex(idStr, "-") + 1
	s := strings.Split(idStr, "-")

	return s[len(s)-1], off + p
}

// lookupContainerID extracts the container ID as a 31-character string length from the full cgroup path.
//
// Parameters:
//   - cgroup: The full cgroup path.
//
// Returns:
//   - string: The container ID as a string of 31 characters.
func lookupContainerID(cgroup string) string {
	subDirs := strings.Split(cgroup, "/")
	lastSubDir := subDirs[len(subDirs)-1]

	container, _ := procsContainerIDOffset(lastSubDir)

	// Ensure the container ID is no longer than BpfContainerIDLength characters.
	if len(container) >= BpfContainerIDLength {
		return container[:BpfContainerIDLength]
	}

	return ""
}
