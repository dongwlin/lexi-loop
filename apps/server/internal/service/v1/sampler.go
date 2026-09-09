// Package v1 实现 service 父包的业务接口，构造函数返回具体指针；
// 请求/结果类型由父包定义，JSON 契约由 Handler 的版本化 DTO 定义
// （docs/backend/structure.md §4.3）。
//
// Service 负责用例事务、外部资源校验、随机抽样以及 Domain / Repo 错误到
// apperr.Error 的映射；复习用例的事务与并发边界见 structure.md §5。
package v1

import (
	"cmp"
	"math"
	mathrand "math/rand/v2"
	"slices"
	"sync"
	"time"
)

// WeightedSampler 把权重用于加权随机不放回抽样（structure.md §6）。
// 抽取算法为 Efraimidis–Spirakis（A-Res）：以 ln(U)/w 为排序键取最大的
// n 个候选，等价于逐次「按权重抽一个并移除」；高权重只表示更容易出现，
// 不保证一定出现，抽样保留随机性（docs/review/algorithm.md §8）。
//
// 随机源在构造时注入以支持确定性测试（固定 source 即可复现抽样序列）；
// MVP 只保留这一个真实实现，不提前声明 Sampler 接口。
type WeightedSampler struct {
	mu   sync.Mutex
	rand *mathrand.Rand
}

// NewWeightedSampler 构造抽样器。src 不得为 nil：生产侧传入以时间派生的
// 种子（NewTimeSeededSource），测试侧传入固定种子（如 mathrand.NewPCG(1, 2)）
// 获得可复现序列。
func NewWeightedSampler(src mathrand.Source) *WeightedSampler {
	return &WeightedSampler{rand: mathrand.New(src)}
}

// NewTimeSeededSource 以当前时间派生随机源种子，供组合根构造生产用
// WeightedSampler 使用。
func NewTimeSeededSource() mathrand.Source {
	seed := uint64(time.Now().UnixNano())
	return mathrand.NewPCG(seed, seed^0x9E3779B97F4A7C15)
}

// Sample 按权重加权随机不放回抽取 n 个候选的下标，权重必须为正
// （domain.UserWord.ReviewWeight 保证下限）。n <= 0 返回空；n 超过候选数
// 时抽取全部（数量截断 min(count, available) 已由调用方完成，此处仅为
// 防御性钳制）。返回的下标互不相同且都在候选范围内。
func (s *WeightedSampler) Sample(weights []float64, n int) []int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n <= 0 || len(weights) == 0 {
		return nil
	}
	if n > len(weights) {
		n = len(weights)
	}

	type keyed struct {
		index int
		key   float64
	}
	keys := make([]keyed, len(weights))
	for i, w := range weights {
		keys[i] = keyed{index: i, key: s.drawKey(w)}
	}
	// 排序键越大越先抽中；全量排序对 MVP 词库规模足够。
	slices.SortFunc(keys, func(a, b keyed) int { return cmp.Compare(b.key, a.key) })

	selected := make([]int, n)
	for i := range selected {
		selected[i] = keys[i].index
	}
	return selected
}

// drawKey 计算单次抽取的排序键 ln(U)/w（A-Res）。U 取自 (0,1) 开区间：
// rand.Float64() 返回 [0,1)，命中 0（概率约 2^-53）时回退为最小正浮点，
// 避免 log(0) 产生 -Inf；非正权重同样按近零权重处理（排序键排在末尾，
// 仅在需要凑满 n 时才可能被抽中），保证抽样器对异常输入不 panic。
func (s *WeightedSampler) drawKey(w float64) float64 {
	if w <= 0 {
		w = math.SmallestNonzeroFloat64
	}
	u := s.rand.Float64()
	if u == 0 {
		u = math.SmallestNonzeroFloat64
	}
	return math.Log(u) / w
}
