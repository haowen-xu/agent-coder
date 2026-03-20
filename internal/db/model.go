package db

import "time"

type SystemInfo struct {
	ID        uint   `gorm:"primaryKey"`
	Key       string `gorm:"size:128;uniqueIndex;not null"`
	Value     string `gorm:"size:1024;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PromptTemplate struct {
	ID         uint   `gorm:"primaryKey"`
	ProjectKey string `gorm:"size:64;not null;index:idx_prompt_project_kind_role,unique"`
	RunKind    string `gorm:"size:16;not null;index:idx_prompt_project_kind_role,unique"`
	AgentRole  string `gorm:"size:16;not null;index:idx_prompt_project_kind_role,unique"`
	Content    string `gorm:"type:text;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
