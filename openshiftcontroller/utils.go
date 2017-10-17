package openshiftcontroller

func roundP(f float64) int {
	return int(f + 0.5)
}

func SplitGroups(data []string, split []*[]string) []*[]string {
	n := len(split)

	for i:=0;i<n;i++ {
		div := float64(len(data))/float64(n)
		start := div*float64(i)
		end := div*float64(i+1)

		p := make([]string, len(data[roundP(start):roundP(end)]))
		split[i] = &p

		copy(*split[i], data[roundP(start):roundP(end)])
	}

	return split
}