// Package ecdict 是 ECDICT CSV 的离线导入适配器与导入用例编排
// （docs/backend/structure.md §8）。解析器属于离线输入适配器，不放进
// Domain 或 Service；运行时读取的只是导入后的本地词典库。
package ecdict

// TODO: parser.go —— CSV 解析与源字段到 Domain 的映射。
