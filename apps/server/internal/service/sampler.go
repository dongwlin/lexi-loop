package service

// TODO: 具体 WeightedSampler —— 把权重用于加权随机不放回抽样。
// MVP 只保留这一真实实现，不提前声明 Sampler 接口；随机源作为构造参数
// 注入以支持确定性测试（docs/backend/structure.md §6）。
