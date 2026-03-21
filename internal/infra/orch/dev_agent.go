package orch

// OrchDevAgent 是 dev 阶段 agent。
type OrchDevAgent struct{ *orchBaseAgent }

// NewOrchDevAgent 构造 OrchDevAgent。
func NewOrchDevAgent(opts AgentOptions) *OrchDevAgent {
	return &OrchDevAgent{orchBaseAgent: newBaseAgent(AgentKindDev, 0, opts)}
}
