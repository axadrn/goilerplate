package chart

import (
	"math"
	"regexp"
	"slices"
	"strconv"
)

// The value-axis algorithms mirror recharts-scale and Recharts 2.15.4
// ChartUtils. Float64 arithmetic follows the existing browser renderer.
// https://github.com/recharts/recharts-scale/blob/v0.4.5/src/getNiceTickValues.js
// https://github.com/recharts/recharts/blob/v2.15.4/src/util/ChartUtils.ts
func getDigitCount(v float64) int {
	if v == 0 {
		return 1
	}
	return int(math.Floor(math.Log10(math.Abs(v)))) + 1
}

func getFormatStep(roughStep float64, allowDecimals bool, correctionFactor int) float64 {
	if roughStep <= 0 {
		return 0
	}
	digitCount := getDigitCount(roughStep)
	digitCountValue := math.Pow10(digitCount)
	stepRatio := roughStep / digitCountValue
	stepRatioScale := 0.05
	if digitCount == 1 {
		stepRatioScale = 0.1
	}
	step := (math.Ceil(stepRatio/stepRatioScale) + float64(correctionFactor)) * stepRatioScale * digitCountValue
	if !allowDecimals {
		return math.Ceil(step)
	}
	return step
}

func calculateStep(min, max float64, tickCount int, allowDecimals bool) (step, tickMin, tickMax float64) {
	rough := (max - min) / float64(tickCount-1)
	if math.IsInf(rough, 0) || math.IsNaN(rough) {
		return 0, 0, 0
	}
	for correction := 0; ; correction++ {
		step = getFormatStep(rough, allowDecimals, correction)
		middle := 0.0
		if min > 0 || max < 0 {
			middle = (min + max) / 2
			middle -= math.Mod(middle, step)
		}
		below, above := math.Ceil((middle-min)/step), math.Ceil((max-middle)/step)
		count := below + above + 1
		if count > float64(tickCount) {
			continue
		}
		if count < float64(tickCount) {
			if max > 0 {
				above += float64(tickCount) - count
			} else {
				below += float64(tickCount) - count
			}
		}
		return step, middle - below*step, middle + above*step
	}
}

func getTickOfSingleValue(value float64, tickCount int, allowDecimals bool) []float64 {
	step, middle := 1.0, value
	if math.Trunc(value) != value && allowDecimals {
		if math.Abs(value) < 1 {
			step = math.Pow10(getDigitCount(value) - 1)
			middle = math.Floor(value/step) * step
		} else if math.Abs(value) > 1 {
			middle = math.Floor(value)
		}
	} else if value == 0 {
		middle = math.Floor(float64(tickCount-1) / 2)
	} else if !allowDecimals {
		middle = math.Floor(value)
	}
	middleIndex := (tickCount - 1) / 2
	ticks := make([]float64, tickCount)
	for i := range ticks {
		ticks[i] = middle + float64(i-middleIndex)*step
	}
	return ticks
}

func getNiceTickValues(domain [2]float64, tickCount int, allowDecimals bool) []float64 {
	min, max := domain[0], domain[1]
	reversed := min > max
	if reversed {
		min, max = max, min
	}
	if min == max {
		return getTickOfSingleValue(min, tickCount, allowDecimals)
	}
	step, lo, hi := calculateStep(min, max, maxInt(tickCount, 2), allowDecimals)
	if step <= 0 {
		return []float64{0}
	}
	var ticks []float64
	for v := lo; v <= hi+0.1*step; v += step {
		ticks = append(ticks, v)
	}
	if reversed {
		slices.Reverse(ticks)
	}
	return ticks
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getTickValuesFixedDomain(domain [2]float64, tickCount int, allowDecimals bool) []float64 {
	min, max := domain[0], domain[1]
	reversed := min > max
	if reversed {
		min, max = max, min
	}
	if math.IsInf(min, -1) || math.IsInf(max, 1) {
		return domain[:]
	}
	if min == max {
		return []float64{min}
	}
	step := getFormatStep((max-min)/float64(maxInt(tickCount, 2)-1), allowDecimals, 0)
	var ticks []float64
	for v := min; v < max-0.99*step; v += step {
		ticks = append(ticks, v)
	}
	ticks = append(ticks, max)
	if reversed {
		slices.Reverse(ticks)
	}
	return ticks
}

var minValueReg = regexp.MustCompile(`^dataMin\s*-\s*([0-9]+(?:\.[0-9]+)?)$`)
var maxValueReg = regexp.MustCompile(`^dataMax\s*\+\s*([0-9]+(?:\.[0-9]+)?)$`)

func parseSpecifiedDomain(specified []any, data [2]float64, allowDataOverflow bool) [2]float64 {
	if len(specified) == 0 {
		return data
	}
	domain := data
	for i, value := range specified {
		if text, ok := value.(string); ok {
			re := minValueReg
			if i == 1 {
				re = maxValueReg
			}
			if match := re.FindStringSubmatch(text); match != nil {
				offset, _ := strconv.ParseFloat(match[1], 64)
				if i == 0 {
					domain[i] = data[i] - offset
				} else {
					domain[i] = data[i] + offset
				}
			}
		} else {
			switch value.(type) {
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
				v, _ := strconv.ParseFloat(str(value), 64)
				if allowDataOverflow {
					domain[i] = v
				} else if i == 0 {
					domain[i] = math.Min(v, data[i])
				} else {
					domain[i] = math.Max(v, data[i])
				}
			}
		}
	}
	return domain
}

func valueDomain(m Model) [2]float64 {
	if m.StackOffset == "expand" {
		return [2]float64{0, 1}
	}
	domain := [2]float64{math.Inf(1), math.Inf(-1)}
	add := func(v float64) { domain[0] = math.Min(domain[0], v); domain[1] = math.Max(domain[1], v) }
	if m.Stacked {
		add(0)
		for i := range m.Labels {
			sum := 0.0
			for _, s := range m.Series {
				if !s.Hidden {
					sum += s.Values[i]
				}
			}
			add(sum)
		}
	} else {
		for _, s := range m.Series {
			if s.Hidden {
				continue
			}
			for i, v := range s.Values {
				if len(s.Gaps) > i && s.Gaps[i] {
					continue
				}
				add(v)
			}
		}
	}
	if math.IsInf(domain[0], 1) {
		return [2]float64{0, 0}
	}
	return domain
}

func setValueScale(m *Model, st *chartState) {
	count := 0
	var specified []any
	var explicitTicks []float64
	allowOverflow := false
	if st.x != nil && len(st.x.Domain) != 0 && len(st.x.Domain) != 2 {
		panic("chart.XAxisProps.Domain must contain exactly two entries")
	}
	if st.y != nil && len(st.y.Domain) != 0 && len(st.y.Domain) != 2 {
		panic("chart.YAxisProps.Domain must contain exactly two entries")
	}
	if st.layout == "vertical" {
		if st.x != nil {
			count = st.x.TickCount
			specified, explicitTicks, allowOverflow = st.x.Domain, st.x.Ticks, st.x.AllowDataOverflow
		}
	} else if st.y != nil {
		count = st.y.TickCount
		specified, explicitTicks, allowOverflow = st.y.Domain, st.y.Ticks, st.y.AllowDataOverflow
	}
	m.TickCount = count
	if count <= 0 {
		count = 5
	}
	if len(specified) == 0 {
		specified = []any{0, "auto"}
	}
	dataDomain := valueDomain(*m)
	// detectReferenceElementsDomain extends the data domain for specified ticks.
	for _, v := range explicitTicks {
		dataDomain[0] = math.Min(dataDomain[0], v)
		dataDomain[1] = math.Max(dataDomain[1], v)
	}
	domain := parseSpecifiedDomain(specified, dataDomain, allowOverflow)
	auto := specified[0] == "auto" || specified[1] == "auto"
	var ticks []float64
	if auto {
		ticks = getNiceTickValues(domain, count, true)
		if m.StackOffset == "expand" && len(explicitTicks) == 0 {
			ticks = make([]float64, count)
			for i := range ticks {
				ticks[i] = float64(i) / float64(count-1)
			}
		}
		domain = [2]float64{ticks[0], ticks[len(ticks)-1]}
	} else {
		ticks = getTickValuesFixedDomain(domain, count, true)
	}
	if len(explicitTicks) > 0 {
		ticks = explicitTicks
	}
	m.AllowDataOverflow = allowOverflow
	m.Domain = domain
	m.Ticks = ticks
	var formatter func(any) string
	if st.layout == "vertical" {
		if st.x != nil {
			formatter = st.x.TickFormatter
		}
	} else if st.y != nil {
		formatter = st.y.TickFormatter
	}
	m.TickLabels = make([]string, len(ticks))
	for i, v := range ticks {
		if formatter != nil {
			m.TickLabels[i] = formatter(v)
		} else {
			m.TickLabels[i] = strconv.FormatFloat(math.Floor(v*1000+0.5)/1000, 'f', -1, 64)
		}
	}
}
