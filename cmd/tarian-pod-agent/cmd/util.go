package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/sirupsen/logrus"
)

func readLabelsFromFile(logger *logrus.Logger, path string) ([]*tarianpb.Label, error) {
	labels := []*tarianpb.Label{}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		idx := strings.Index(line, "=")

		if idx < 0 {
			continue
		}

		key := line[:idx]
		value := strings.Trim(line[idx+1:], "\"")
		logger.Debugf("Read label from file: %s=%s", key, value)

		labels = append(labels, &tarianpb.Label{Key: key, Value: value})
	}

	if len(labels) == 0 {
		return nil, fmt.Errorf("no labels found in file")
	}
	return labels, nil
}
