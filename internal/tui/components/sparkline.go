package components

var sparkBlocks = []rune{'\u2581', '\u2582', '\u2583', '\u2584', '\u2585', '\u2586', '\u2587', '\u2588'}

// Sparkline renders a sparkline from data within maxWidth characters.
func Sparkline(data []int, maxWidth int) string {
	if len(data) == 0 {
		return ""
	}

	if len(data) > maxWidth {
		sampled := make([]int, maxWidth)
		ratio := float64(len(data)) / float64(maxWidth)
		for i := range sampled {
			idx := int(float64(i) * ratio)
			if idx >= len(data) {
				idx = len(data) - 1
			}
			sampled[i] = data[idx]
		}
		data = sampled
	}

	minVal, maxVal := data[0], data[0]
	for _, v := range data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	result := make([]rune, len(data))
	span := maxVal - minVal
	for i, v := range data {
		if span == 0 {
			result[i] = sparkBlocks[len(sparkBlocks)/2]
		} else {
			idx := (v - minVal) * (len(sparkBlocks) - 1) / span
			result[i] = sparkBlocks[idx]
		}
	}

	return string(result)
}
