package domains

// DeploymentClosure represents the closure-table rows for deployment hierarchy
type DeploymentClosure struct {
	AncestorID   string `gorm:"type:varchar(255);index:idx_deploy_closure_ancestor_descendant,priority:1;not null" json:"ancestorId"`
	DescendantID string `gorm:"type:varchar(255);index:idx_deploy_closure_ancestor_descendant,priority:2;not null" json:"descendantId"`
	Depth        int    `gorm:"not null" json:"depth"`
}

// TableName sets explicit table name to match conventional plural
func (DeploymentClosure) TableName() string {
	return "deployment_closures"
}
