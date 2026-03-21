package orch

// OrchReviewAgent 是 review 阶段 agent。
type OrchReviewAgent struct{ *orchBaseAgent }

// NewOrchReviewAgent 构造 OrchReviewAgent。
func NewOrchReviewAgent(opts AgentOptions) *OrchReviewAgent {
	return &OrchReviewAgent{orchBaseAgent: newBaseAgent(AgentKindReview, 0, opts)}
}
