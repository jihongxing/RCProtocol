package repo

// repo 层直接操作 SQL，不做独立 repo 层单元测试。
// SQL 正确性通过 handler 层集成测试（mock repo 接口）间接覆盖。
//
// 风险说明：若 SQL 语法/逻辑有误（如字段顺序、COALESCE 行为等），
// handler 层 mock 测试无法发现，需在集成环境中验证。
