package cmd

import (
	"os"
	"testing"

	"github.com/kube-tarian/tarian/pkg/log"
	"github.com/kube-tarian/tarian/pkg/tarianpb"
	"github.com/stretchr/testify/assert"
)

func TestReadLabelsFromFile(t *testing.T) {
	file := createTestLabelsFile(t, false)
	defer os.Remove(file.Name())
	tests := []struct {
		name           string
		fileName       string
		expectedErr    string
		expectedLabels []*tarianpb.Label
	}{
		{
			name:     "Valid Labels File",
			fileName: file.Name(),
			expectedLabels: []*tarianpb.Label{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
				{Key: "key3", Value: "value3"},
			},
		},
		{
			name:        "Invalid Labels File",
			fileName:    "nonexistent_labels.txt",
			expectedErr: "failed to open file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labels, err := readLabelsFromFile(log.GetLogger(), tt.fileName)

			if tt.expectedErr != "" {
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Len(t, labels, len(tt.expectedLabels))

				for i, expectedLabel := range tt.expectedLabels {
					assert.Equal(t, expectedLabel.Key, labels[i].Key)
					assert.Equal(t, expectedLabel.Value, labels[i].Value)
				}
			}
		})
	}
}

func createTestLabelsFile(t *testing.T, empty bool) *os.File {
	file, _ := os.CreateTemp("", "test_labels_*.txt")
	if !empty {
		content := []byte("key1=\"value1\"\nkey2=\"value2\"\nkey3=\"value3\"\n")
		_, err := file.Write(content)
		assert.NoError(t, err)
		err = file.Close()
		assert.NoError(t, err)
	}
	return file
}
