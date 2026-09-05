package domain

// TODO: 定义领域 sentinel / typed errors（如「释义为空」「item 已作答」等），
// 供 Service 通过 errors.Is / errors.As 识别后映射为 apperr.Error。
// 示例：var ErrMeaningRequired = errors.New("domain: review meaning required")
