package service

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Sample 的确定性依赖注入的随机源：固定种子复现同一序列，
// 不同种子产生不同序列（docs/backend/structure.md §6）。
func TestWeightedSampler_Sample(t *testing.T) {
	t.Parallel()

	t.Run("固定种子可复现抽样序列", func(t *testing.T) {
		t.Parallel()
		weights := []float64{1, 3, 6, 2, 5, 4}

		first := NewWeightedSampler(rand.NewPCG(1, 2)).Sample(weights, 3)
		second := NewWeightedSampler(rand.NewPCG(1, 2)).Sample(weights, 3)

		assert.Equal(t, first, second)
	})

	t.Run("不同种子产生不同序列", func(t *testing.T) {
		t.Parallel()
		weights := make([]float64, 64)
		for i := range weights {
			weights[i] = float64(i + 1)
		}

		first := NewWeightedSampler(rand.NewPCG(1, 2)).Sample(weights, 8)
		second := NewWeightedSampler(rand.NewPCG(3, 4)).Sample(weights, 8)

		assert.NotEqual(t, first, second)
	})

	t.Run("结果下标互不相同且在候选范围内", func(t *testing.T) {
		t.Parallel()
		weights := []float64{1.5, 0.7, 9.2, 3.3, 2.2}

		selected := NewWeightedSampler(rand.NewPCG(42, 43)).Sample(weights, 5)

		require.Len(t, selected, 5)
		slices.Sort(selected)
		assert.Equal(t, []int{0, 1, 2, 3, 4}, selected, "抽满全部候选时应恰好是全体下标")
	})

	t.Run("n 超过候选数时钳制为全部候选", func(t *testing.T) {
		t.Parallel()
		selected := NewWeightedSampler(rand.NewPCG(1, 2)).Sample([]float64{1, 2}, 10)
		assert.Len(t, selected, 2)
	})

	t.Run("n 非正或无候选时返回空", func(t *testing.T) {
		t.Parallel()
		sampler := NewWeightedSampler(rand.NewPCG(1, 2))
		assert.Nil(t, sampler.Sample([]float64{1, 2}, 0))
		assert.Nil(t, sampler.Sample([]float64{1, 2}, -1))
		assert.Nil(t, sampler.Sample(nil, 3))
	})

	t.Run("高权重候选以显著更高概率被抽中", func(t *testing.T) {
		t.Parallel()
		// 权重 6 与 1 的单次抽取概率应为 6/7 ≈ 85.7% 与 1/7 ≈ 14.3%；
		// 10000 次独立单抽的频率偏离不应超过 ±5%（统计断言放宽容忍带，
		// 避免偶发抖动导致假失败）。
		const draws = 10000
		heavy := 0
		for i := 0; i < draws; i++ {
			selected := NewWeightedSampler(rand.NewPCG(uint64(i), uint64(i)+1)).Sample([]float64{1, 6}, 1)
			if selected[0] == 1 {
				heavy++
			}
		}
		assert.InDelta(t, 6.0/7.0, float64(heavy)/draws, 0.05)
	})
}
