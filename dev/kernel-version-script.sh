#!/bin/bash
    
# Get the Linux kernel version string
KERNEL_VERSION=$(uname -r)
    
# Extract major and minor version
LINUX_VERSION_MAJOR=$(echo "$KERNEL_VERSION" | cut -d'.' -f1)
LINUX_VERSION_MINOR=$(echo "$KERNEL_VERSION" | cut -d'.' -f2)
LINUX_VERSION_PATCH=$(echo "$kernel_version" | cut -d'.' -f3)
    
# Export major and minor version as environment variables
export LINUX_VERSION_MAJOR
export LINUX_VERSION_MINOR
export LINUX_VERSION_PATCH

echo "Kernel major, minor and patch version set as environment variable: $LINUX_VERSION_MAJOR, $LINUX_VERSION_MINOR, $LINUX_VERSION_PATCH"
