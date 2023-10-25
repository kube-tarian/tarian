package add

import (
	"strings"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
)

func matchLabelsFromString(strLabels []string) []*tarianpb.MatchLabel {
	if strLabels == nil {
		return nil
	}

	labels := []*tarianpb.MatchLabel{}

	for _, s := range strLabels {
		idx := strings.Index(s, "=")

		if idx < 0 || idx == len(s)-1 {
			continue
		}

		key := s[:idx]
		value := strings.Trim(s[idx+1:], "\"")

		labels = append(labels, &tarianpb.MatchLabel{Key: key, Value: value})
	}

	if len(labels) == 0 {
		return nil
	}
	return labels
}
