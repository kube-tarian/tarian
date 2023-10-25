package get

import (
	"strings"

	"github.com/kube-tarian/tarian/pkg/tarianpb"
)

func matchLabelsToString(labels []*tarianpb.MatchLabel) string {
	if len(labels) == 0 {
		return ""
	}

	str := strings.Builder{}
	str.WriteString("matchLabels:")

	for i, l := range labels {
		str.WriteString(l.GetKey())
		str.WriteString("=")
		str.WriteString(l.GetValue())

		if i < len(labels)-1 {
			str.WriteString(",")
		}
	}

	return str.String()
}
