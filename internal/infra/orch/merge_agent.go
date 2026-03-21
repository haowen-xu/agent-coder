package orch

// OrchMergeAgent 是 merge 阶段 agent。
// per-project 限流使用 queue 的 maxProjectWorkers。
type OrchMergeAgent struct{ *orchBaseAgent }

// NewOrchMergeAgent 构造 OrchMergeAgent。
func NewOrchMergeAgent(opts AgentOptions) *OrchMergeAgent {
	return &OrchMergeAgent{orchBaseAgent: newBaseAgent(AgentKindMerge, -1, opts)}
}
