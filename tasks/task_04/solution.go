package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	count := len(nums)
	if count < 2 {
		return Stats{}
	}
	stats := Stats{
		Count: count - 1,
		Min:   0,
		Max:   0,
		Sum:   0,
	}
	for i := range count {
		if i == 0 {
			continue
		}
		diff := nums[i] - nums[i-1]
		if i == 1 {
			stats.Max = diff
			stats.Min = diff
		}
		stats.Sum += diff
		if diff > stats.Max {
			stats.Max = diff
		} else if diff < stats.Min {
			stats.Min = diff
		}
	}
	return stats
}
