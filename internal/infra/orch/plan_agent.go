package orch

// OrchPlanAgent 是 plan 阶段 agent。
type OrchPlanAgent struct{ *orchBaseAgent }

// NewOrchPlanAgent 构造 OrchPlanAgent。
func NewOrchPlanAgent(opts AgentOptions) *OrchPlanAgent {
	return &OrchPlanAgent{orchBaseAgent: newBaseAgent(AgentKindPlan, 0, opts)}
}
